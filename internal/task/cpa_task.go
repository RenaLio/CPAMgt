package task

import (
	"context"
	"cpamgt/internal/external/cpa"
	"cpamgt/internal/model"
	"cpamgt/internal/pkg/log"
	"cpamgt/internal/service"
	"cpamgt/pkg/httpclient"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultCpaPoolSize = 3
)

type CpaTask struct {
	log      *log.Logger
	svc      service.CpaAccountService
	tokenSvc service.TokenAccountService
}

func (c *CpaTask) CurrentStats() (any, error) {
	return nil, nil
}

func NewCpaTask(
	log *log.Logger,
	svc service.CpaAccountService,
	tokenSvc service.TokenAccountService,
) *CpaTask {
	return &CpaTask{
		log:      log,
		svc:      svc,
		tokenSvc: tokenSvc,
	}
}

func getCpaPoolSize() int {
	poolSize := os.Getenv("CPA_POOL_SIZE")
	if poolSize == "" {
		return defaultCpaPoolSize
	}
	poolNum, err := strconv.Atoi(poolSize)
	if err != nil {
		return defaultCpaPoolSize
	}
	if poolNum < 1 {
		return defaultCpaPoolSize
	}
	return poolNum
}

func (c *CpaTask) Name() string {
	return CpaTaskName
}

func (c *CpaTask) Run(ctx context.Context) error {
	return c.Do(ctx)
}

func (c *CpaTask) Log(ctx context.Context) *log.Logger {
	return c.log.FromContext(ctx)
}

// 添加可用，删除不可用的
// 删除数据库中的记录

func (c *CpaTask) Do(ctx context.Context) error {
	config, err := c.svc.GetCpaConfig(ctx)
	if err != nil {
		return err
	}
	if !config.CpaEnable {
		return nil
	}

	httpc := httpclient.GetDefaultClient()
	cpaClient := cpa.NewCpaClient(config.CpaUrl, config.CpaToken, "", httpc)

	tokenAccounts, err := c.tokenSvc.ListAll(ctx)
	if err != nil {
		return err
	}

	cpaAuthFiles, err := cpaClient.ListAuthFiles(ctx)
	if err != nil {
		return err
	}

	remoteAuthFiles := make(map[string]bool)
	for _, item := range cpaAuthFiles.Files {
		remoteAuthFiles[item.Name] = true
	}

	taskCh := make(chan *model.TokenAccount, getCpaPoolSize()+1)
	wg := new(sync.WaitGroup)

	go func() {
		for _, item := range tokenAccounts {
			taskCh <- &item
		}
		close(taskCh)
	}()

	poolSize := getCpaPoolSize()
	for _ = range poolSize {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range taskCh {
				if item.Status == model.TokenAccountStatusAvailable {
					data := make(map[string]string)
					data["id_token"] = item.IDToken
					data["access_token"] = item.AccessToken
					data["refresh_token"] = item.RefreshToken
					data["account_id"] = item.AccountID
					data["last_refresh"] = time.Now().Format(time.RFC3339)
					data["email"] = item.Email
					data["type"] = item.AccountType
					data["expired"] = item.ExpiredAt.Format(time.RFC3339)
					// 添加
					fileName := normalizeFileName(*item)
					fileContent, err := json.Marshal(data)
					if err != nil {
						c.Log(ctx).Error("marshal token account failed", "id", item.ID, "err", err)
						continue
					}
					err = cpaClient.UploadAuthFile(ctx, fileName, fileContent)
					if err != nil {
						c.Log(ctx).Error("upload auth file failed", "id", item.ID, "err", err)
						continue
					}
					continue
				}
				if item.CpaDelFlag == 0 {
					continue
				}
				fileName := normalizeFileName(*item)
				if !remoteAuthFiles[fileName] {
					continue
				}
				err = cpaClient.DeleteAuthFile(ctx, fileName)
				if err != nil {
					c.Log(ctx).Error("delete auth file failed", "id", item.ID, "err", err)
					if !errors.Is(err, cpa.ErrNotFound) {
						continue
					}
				}
				if item.Status == model.TokenAccountStatusAuthExpired {
					// 下次就不执行删除操作了
					item.CpaDelFlag = 0
					c.tokenSvc.Update(ctx, &service.UpdateTokenAccountInput{
						ID:         item.ID,
						CpaDelFlag: new(uint8(0)),
					})
				}
			}
		}()
	}

	wg.Wait()
	return nil
}

func normalizeFileName(a model.TokenAccount) string {
	return strings.ToLower(fmt.Sprintf("%s_%s.json", a.AccountType, a.Email))
}
