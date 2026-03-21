package httpclient

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	MaxRedirects          = 10
	DefaultTimeout        = 180 * time.Second
	DefaultMaxIdleConns   = 256
	DefaultMaxIdlePerHost = 64

	CustomPoolMaxEntries = 1024
	CustomPoolTTL        = 60 * time.Minute

	PoolMaintenanceInterval = 30 * time.Second // 周期保底清理
)

type poolEntry struct {
	client   *http.Client
	lastUsed int64 // UnixNano
}

var (
	initOnce      sync.Once
	httpclient    *http.Client
	defaultConfig Config
	customPool    sync.Map // map[string]*poolEntry

	customPoolSize int64
	poolEvictMu    sync.Mutex // 保护淘汰逻辑

	maintenanceOnce sync.Once
	maintenanceWake chan struct{} // 非阻塞触发清理

	poolMu sync.RWMutex // 统一锁
)

// Config 客户端配置参数
type Config struct {
	Proxy             string
	Timeout           time.Duration
	FirstBytesTimeout time.Duration
	MaxIdleConns      int
	MaxIdlePerHost    int
	SkipVerify        bool
}

// Init 初始化全局默认客户端
func Init() {
	initOnce.Do(func() {
		defaultConfig = Config{
			Proxy:          "",
			Timeout:        DefaultTimeout,
			MaxIdleConns:   DefaultMaxIdleConns,
			MaxIdlePerHost: DefaultMaxIdlePerHost,
			SkipVerify:     false,
		}
		httpclient, _ = createClient(defaultConfig)
		startPoolMaintenance()
	})
}

// checkRedirect 重定向控制
func checkRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= MaxRedirects {
		return fmt.Errorf("stopped after %d redirects", MaxRedirects)
	}
	return nil
}

// GetDefaultClient 获取全局默认的 Client
func GetDefaultClient() *http.Client {
	Init()
	return httpclient
}

// GetDefineClient 获取自定义配置的 Client (自带连接池缓存)
func GetDefineClient(config Config) (*http.Client, error) {
	Init() // 防呆设计：确保基础依赖已初始化
	poolMu.RLock()
	defer poolMu.RUnlock()

	config = normalizeConfigForPool(config)
	poolKey := buildPoolKey(config)
	now := time.Now().UnixNano()

	// 1) 先读缓存
	if v, ok := customPool.Load(poolKey); ok {
		e := v.(*poolEntry)
		atomic.StoreInt64(&e.lastUsed, now)
		return e.client, nil
	}

	// 2) 创建
	c, err := createClient(config)
	if err != nil {
		return c, err
	}

	newEntry := &poolEntry{
		client:   c,
		lastUsed: now,
	}

	// 3) 并发去重
	actual, loaded := customPool.LoadOrStore(poolKey, newEntry)
	if loaded {
		// 并发下重复创建的 client 没入池，回收掉
		c.CloseIdleConnections()

		e := actual.(*poolEntry)
		atomic.StoreInt64(&e.lastUsed, now)
		return e.client, nil
	}

	// 真实新增了一个 Client
	atomic.AddInt64(&customPoolSize, 1)

	// 4) 惰性清理 (非阻塞设计)
	triggerPoolMaintenance()

	return c, nil
}

func startPoolMaintenance() {
	maintenanceOnce.Do(func() {
		maintenanceWake = make(chan struct{}, 1)

		go func() {
			ticker := time.NewTicker(PoolMaintenanceInterval)
			defer ticker.Stop()

			for {
				select {
				case <-maintenanceWake:
				// 主动保底：定期清理，避免长期漏清
				case <-ticker.C:
				}

				cleanupCustomPoolExpired()
				evictCustomPoolIfNeeded()
			}
		}()
	})
}

func triggerPoolMaintenance() {
	select {
	case maintenanceWake <- struct{}{}:
	default:
		// 通道满则忽略，表示已有待处理任务，避免重复触发
	}
}

// cleanupCustomPoolExpired 清理过期连接
func cleanupCustomPoolExpired() {
	// 使用 TryLock，如果有其他人正在扫地，我直接下班，避免 CPU 惊群效应
	if !poolEvictMu.TryLock() {
		return
	}
	defer poolEvictMu.Unlock()

	cutoff := time.Now().Add(-CustomPoolTTL).UnixNano()

	customPool.Range(func(key, value any) bool {
		e := value.(*poolEntry)
		if atomic.LoadInt64(&e.lastUsed) < cutoff {
			if actual, ok := customPool.LoadAndDelete(key); ok {
				if ae, ok := actual.(*poolEntry); ok && ae.client != nil {
					ae.client.CloseIdleConnections()
				}
				atomic.AddInt64(&customPoolSize, -1)
			}
		}
		return true
	})
}

// evictCustomPoolIfNeeded 超出容量时 LRU 淘汰
func evictCustomPoolIfNeeded() {
	// 提前判断，没超标直接退出
	if atomic.LoadInt64(&customPoolSize) <= CustomPoolMaxEntries {
		return
	}

	// TryLock 防止积压
	if !poolEvictMu.TryLock() {
		return
	}
	defer poolEvictMu.Unlock()

	// 拿到锁后 Double Check
	currentSize := atomic.LoadInt64(&customPoolSize)
	if currentSize <= CustomPoolMaxEntries {
		return
	}

	// 需要删掉的元素数量
	needRemove := int(currentSize - CustomPoolMaxEntries)

	// 收集所有 key 和时间戳
	type item struct {
		key      any
		lastUsed int64
	}
	var items []item

	customPool.Range(func(key, value any) bool {
		e := value.(*poolEntry)
		items = append(items, item{
			key:      key,
			lastUsed: atomic.LoadInt64(&e.lastUsed),
		})
		return true
	})

	// 按照 lastUsed 升序排序 (最旧的排在前面)
	sort.Slice(items, func(i, j int) bool {
		return items[i].lastUsed < items[j].lastUsed
	})

	// 批量干掉最旧的 N 个元素，避免 O(N) 嵌套执行
	for i := 0; i < needRemove && i < len(items); i++ {
		if actual, ok := customPool.LoadAndDelete(items[i].key); ok {
			if ae, ok := actual.(*poolEntry); ok && ae.client != nil {
				ae.client.CloseIdleConnections()
			}
			atomic.AddInt64(&customPoolSize, -1)
		}
	}
}

// Close 关闭所有 Client 的空闲连接，释放 goroutine 和端口
func Close() {
	if httpclient != nil {
		httpclient.CloseIdleConnections()
	}
	poolMu.Lock()
	defer poolMu.Unlock()

	poolEvictMu.Lock() // 锁住，防止 close 期间有新请求触发异步清理
	defer poolEvictMu.Unlock()

	customPool.Range(func(key, value any) bool {
		if actual, ok := customPool.LoadAndDelete(key); ok {
			if e, ok := actual.(*poolEntry); ok && e.client != nil {
				e.client.CloseIdleConnections()
			}
		}
		return true
	})

	atomic.StoreInt64(&customPoolSize, 0)

	// 注意: Clear 方法需要 Go 1.21+。如果老版本报错，删掉这行即可，因为上面的 Range 已经清空了。
	customPool.Clear()
}

// ================== 内部辅助函数 ==================

func createClient(config Config) (*http.Client, error) {
	transport := &http.Transport{
		ForceAttemptHTTP2: true,
		Proxy:             http.ProxyFromEnvironment,

		MaxIdleConns:        config.MaxIdleConns,
		MaxIdleConnsPerHost: config.MaxIdlePerHost,
		MaxConnsPerHost:     config.MaxIdlePerHost << 1,
		IdleConnTimeout:     90 * time.Second,

		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	if config.FirstBytesTimeout > 0 {
		transport.ResponseHeaderTimeout = config.FirstBytesTimeout
	}

	if config.Proxy != "" {
		if proxyURL, err := url.Parse(config.Proxy); err != nil {
			return nil, err
		} else {
			transport.Proxy = http.ProxyURL(proxyURL)
		}

	}

	if config.SkipVerify {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	actualTimeout := config.Timeout
	if actualTimeout < 0 {
		actualTimeout = 0
	}

	return &http.Client{
		Transport:     transport,
		Timeout:       actualTimeout,
		CheckRedirect: checkRedirect,
	}, nil
}

func normalizeConfigForPool(c Config) Config {
	if c.Timeout == 0 {
		c.Timeout = DefaultTimeout
	} else if c.Timeout < 0 {
		c.Timeout = -1
	}

	if c.FirstBytesTimeout < 0 {
		c.FirstBytesTimeout = 0
	}

	if c.MaxIdleConns <= 0 {
		c.MaxIdleConns = DefaultMaxIdleConns
	}
	if c.MaxIdlePerHost <= 0 {
		c.MaxIdlePerHost = DefaultMaxIdlePerHost
	}

	c.Proxy = normalizeProxyForKey(c.Proxy)
	return c
}

func buildPoolKey(c Config) string {
	proxyKey := c.Proxy
	if proxyKey == "" {
		proxyKey = "__ENV_PROXY__"
	}

	return fmt.Sprintf(
		"proxy=%s|timeout=%d|firstBytes=%d|maxIdle=%d|maxIdleHost=%d|skipVerify=%t",
		proxyKey,
		int64(c.Timeout),
		int64(c.FirstBytesTimeout),
		c.MaxIdleConns,
		c.MaxIdlePerHost,
		c.SkipVerify,
	)
}

func normalizeProxyForKey(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}

	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	u.Fragment = ""
	u.RawFragment = ""
	if u.Path == "/" {
		u.Path = ""
		u.RawPath = ""
	}
	u.RawQuery = ""
	return u.String()
}
