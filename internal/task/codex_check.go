package task

import (
	"context"
	"cpamgt/internal/external/auth/codex"
	"cpamgt/internal/model"
	"cpamgt/internal/pkg/log"
	"cpamgt/internal/service"
	"cpamgt/pkg/httpclient"
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// 1.获取全部文件
// 2.1如果已经失效。跳过
// 2.2 access_token 是否过期，如果小于6天，刷新，标志位 - refresh_token_expired
// 3 检查usage，如果已经是用完了，并且没到刷新余额的窗口，跳过 标志位 access_token_expired
// 3.1 usage获取失败，如果 at + rt ,整体失效
// 3.2 usage获取成功，判断是否还存在额度
// 4 更新数据库

const (
	PcodexPoolSize = 2
)

type CodexCheckTask struct {
	log *log.Logger
	svc service.TokenAccountService
}

func (t *CodexCheckTask) CurrentStats() (any, error) {
	return nil, nil
}

func NewCodexCheckTask(log *log.Logger, svc service.TokenAccountService) *CodexCheckTask {
	return &CodexCheckTask{log: log, svc: svc}
}

func (t *CodexCheckTask) Name() string {
	return CodexCheckTaskName
}

func getPoolSize() int {
	poolSize := os.Getenv("CODEX_CHECK_POOL_SIZE")
	if poolSize == "" {
		return PcodexPoolSize
	}
	poolNum, err := strconv.Atoi(poolSize)
	if err != nil {
		return PcodexPoolSize
	}
	if poolNum < 1 {
		return PcodexPoolSize
	}
	return poolNum
}

func (t *CodexCheckTask) Run(ctx context.Context) error {
	accounts, err := t.svc.ListAll(ctx)
	if err != nil {
		return err
	}
	taskCh := make(chan *model.TokenAccount, PcodexPoolSize+1)

	wg := new(sync.WaitGroup)
	// 1:M
	go func() {
		for _, account := range accounts {
			taskCh <- &account
		}
		close(taskCh)
	}()

	poolSize := getPoolSize()

	for _ = range poolSize {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for account := range taskCh {
				if err := t.CheckAccount(ctx, *account); err != nil {
					t.Log(ctx).Warn("failed to check account", "err", err)
				}
			}
		}()
	}
	wg.Wait()

	return nil
}

func (t *CodexCheckTask) Log(ctx context.Context) *log.Logger {
	return t.log.FromContext(ctx)
}

func (t *CodexCheckTask) CheckAccount(ctx context.Context, account model.TokenAccount) error {
	if account.Status == model.TokenAccountStatusAuthExpired {
		return nil
	}
	if account.AccountType != "codex" {
		return nil
	}

	fnTrace := make([]string, 0)

	httpc := httpclient.GetDefaultClient()

	extra := make(map[string]any)
	extra["start_status"] = account.Status

	refrefreshTokenExpired := false
	codexClient := codex.NewClient(account.AccessToken, account.RefreshToken, account.AccountID, httpc)

	// 提前6天刷新access_token
	if account.ExpiredAt.Add(6 * time.Hour * 24).Before(time.Now()) {
		trace := "refresh_token"

		data, err := codexClient.RefreshTokens(ctx)
		if err != nil {
			extra["refresh_token_error"] = err.Error()
			if errors.Is(err, codex.ErrTokenExpired) {
				refrefreshTokenExpired = true
			}
			t.Log(ctx).Warn("failed to refresh token", "err", err)
			trace += "[fail]"
		} else {
			account.IDToken = data.IDToken
			account.AccessToken = data.AccessToken
			account.RefreshToken = data.RefreshToken
			account.Email = data.Email
			account.ExpiredAt = data.Expire
			account.LastRefresh = new(time.Now())
			trace += "[success]"
		}
		fnTrace = append(fnTrace, trace)
	}

	// usage 检查
	accessTokenExpired := false
	notRemainingUsage := false
	if account.Status == model.TokenAccountStatusQuotaExhausted {
		notRemainingUsage = true
	}
	if !(account.QuotaRefreshTime != nil && account.Status == model.TokenAccountStatusQuotaExhausted && account.QuotaRefreshTime.After(time.Now())) {
		trace := "usage"
		data, err := codexClient.GetUsage(ctx)
		if err != nil {
			extra["usage_error"] = err.Error()
			if errors.Is(err, codex.ErrTokenExpired) {
				accessTokenExpired = true
				extra["access_token_expired"] = true
			}
			t.Log(ctx).Warn("failed to get usage", "err", err)
			trace += "[fail]"
		} else {
			account.QuotaRefreshTime = new(time.Unix(data.RateLimit.PrimaryWindow.ResetAt, 0))
			account.Percent = int64(data.RateLimit.PrimaryWindow.UsedPercent)
			trace += "[success]"
			if account.Percent >= 100 {
				account.Percent = 100
				notRemainingUsage = true
				trace += "[not]"
			} else {
				notRemainingUsage = false
			}
		}
		fnTrace = append(fnTrace, trace)
	}

	// 更新数据库
	// 更新数据库
	trace := strings.Join(fnTrace, "--")
	extra["trace"] = trace
	extra["rt_expired"] = refrefreshTokenExpired
	extra["at_expired"] = accessTokenExpired
	extra["not_remaining_usage"] = notRemainingUsage
	jbyte, _ := json.Marshal(extra)
	account.Extra = jbyte
	if refrefreshTokenExpired && accessTokenExpired {
		account.Status = model.TokenAccountStatusAuthExpired
	} else if accessTokenExpired {
		account.Status = model.TokenAccountStatusQuotaExhausted
		account.ExpiredAt = time.Now().Add(24 * time.Hour * -10)
		account.Percent = 100
		account.QuotaRefreshTime = new(time.Now())
	} else if notRemainingUsage {
		account.Status = model.TokenAccountStatusQuotaExhausted
	} else {
		account.Status = model.TokenAccountStatusAvailable
	}

	_, err := t.svc.Update(ctx, &service.UpdateTokenAccountInput{
		ID:               account.ID,
		IDToken:          &account.IDToken,
		AccessToken:      &account.AccessToken,
		RefreshToken:     &account.RefreshToken,
		AccountID:        &account.AccountID,
		Email:            &account.Email,
		AccountType:      &account.AccountType,
		ExpiredAt:        &account.ExpiredAt,
		Status:           &account.Status,
		Percent:          &account.Percent,
		QuotaRefreshTime: account.QuotaRefreshTime,
		Extra:            &account.Extra,
	})
	if err != nil {
		return err
	}
	return nil
}
