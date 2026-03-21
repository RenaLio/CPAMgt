// Package httpclient provides a highly concurrent, memory-safe, and auto-scaling HTTP client pool.
// It supports dynamic proxy switching, LRU/TTL eviction, and graceful shutdown.
//
// 提供一个高并发、内存安全且支持自动扩缩容的 HTTP 客户端池。
// 支持动态代理切换、LRU/TTL 自动淘汰以及优雅停机。
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
	MaxRedirects          = 10                // Maximum number of redirects allowed | 最大允许重定向次数
	DefaultTimeout        = 180 * time.Second // Default request timeout | 默认整体请求超时时间
	DefaultMaxIdleConns   = 256               // Default max idle connections in total | 默认最大空闲连接总数
	DefaultMaxIdlePerHost = 64                // Default max idle connections per host | 默认单域名的最大空闲连接数

	CustomPoolMaxEntries = 1024             // Max capacity of the client pool (LRU threshold) | 客户端池最大容量 (触发 LRU 淘汰的阈值)
	CustomPoolTTL        = 60 * time.Minute // Time-To-Live for idle clients in the pool | 池中空闲客户端的存活时间 (触发 TTL 过期清理)
)

// poolEntry wraps the http.Client with a timestamp for LRU and TTL eviction mechanisms.
// 包装 http.Client 并附加时间戳，用于支撑 LRU 和 TTL 淘汰机制。
type poolEntry struct {
	client   *http.Client
	lastUsed int64 // UnixNano timestamp of the last usage | 最后一次使用的时间戳 (纳秒)
}

var (
	initOnce      sync.Once
	httpclient    *http.Client
	defaultConfig Config
	customPool    sync.Map // thread-safe map storing custom clients: map[string]*poolEntry | 存储自定义客户端的并发安全字典

	customPoolSize int64      // Atomic counter for the current size of customPool | 记录 customPool 当前大小的原子计数器
	poolEvictMu    sync.Mutex // Mutex to prevent multiple goroutines from running eviction simultaneously | 防止多个协程同时执行淘汰逻辑的互斥锁

	// Advanced concurrency controls for background maintenance (Event Coalescing)
	// 用于后台清理协程的高级并发控制标志 (事件合并机制)
	maintenanceRunning uint32 // 0 = idle, 1 = running | 当前是否有清理协程正在运行
	maintenancePending uint32 // 0 = no work, 1 = work pending | 当前是否有待处理的清理任务

	// poolMu synchronizes GetDefineClient (RLock) and Close (Lock) for graceful shutdown.
	// 读写锁：同步 GetDefineClient (加读锁) 和 Close (加写锁)，确保优雅停机时不会残留僵尸连接。
	poolMu sync.RWMutex
)

// Config represents the parameters for customizing an HTTP client.
// 客户端配置参数
type Config struct {
	Proxy             string        // Custom proxy URL (empty uses env vars) | 自定义代理地址，为空则默认使用系统环境变量代理
	Timeout           time.Duration // Total timeout (0 = default 3m, -1 = no timeout) | 整体请求超时 (传 0 使用默认 3min，传 -1 表示永不超时)
	FirstBytesTimeout time.Duration // Time to wait for response headers | 首字节超时 (等待响应头的时间，不设置则不处理)
	MaxIdleConns      int           // Max idle connections across all hosts | 最大空闲连接总数
	MaxIdlePerHost    int           // Max idle connections per host | 每个 Host 的最大空闲连接数
	SkipVerify        bool          // Skip TLS certificate verification | 是否跳过 TLS 证书校验
}

// Init initializes the global default client. Safe to call multiple times.
// 初始化全局默认客户端。多次调用是安全的。
func Init() {
	initOnce.Do(func() {
		defaultConfig = Config{
			Proxy:          "", // Empty means using environment proxies | 为空代表使用系统 env 代理
			Timeout:        DefaultTimeout,
			MaxIdleConns:   DefaultMaxIdleConns,
			MaxIdlePerHost: DefaultMaxIdlePerHost,
			SkipVerify:     false,
		}
		httpclient, _ = createClient(defaultConfig)
	})
}

// checkRedirect prevents infinite redirect loops.
// 拦截过多的重定向操作，防止死循环。
func checkRedirect(_ *http.Request, via []*http.Request) error {
	if len(via) >= MaxRedirects {
		return fmt.Errorf("stopped after %d redirects", MaxRedirects)
	}
	return nil
}

// GetDefaultClient returns the global default http.Client.
// 获取全局默认的直连 Client。
func GetDefaultClient() *http.Client {
	Init()
	return httpclient
}

// GetDefineClient retrieves or creates an http.Client based on the given Config.
// It caches the clients to reuse underlying TCP connection pools and prevent port exhaustion.
//
// 获取自定义配置的 Client (自带连接池缓存机制)。
// 它会缓存相同配置的客户端，以复用底层的 TCP Keep-Alive 连接，防止协程和端口泄露。
func GetDefineClient(config Config) (*http.Client, error) {
	Init() // Fail-safe: ensure dependencies are initialized | 防呆设计：确保基础依赖已初始化

	// RLock allows concurrent gets, but blocks if Close() is being executed.
	// 读锁允许极高并发的获取操作，但如果在执行 Close() 优雅停机时，会被阻塞。
	poolMu.RLock()
	defer poolMu.RUnlock()

	config = normalizeConfigForPool(config)
	poolKey := buildPoolKey(config)
	now := time.Now().UnixNano()

	// 1) Fast path: Cache hit | 优先尝试从缓存命中
	if v, ok := customPool.Load(poolKey); ok {
		e := v.(*poolEntry)
		atomic.StoreInt64(&e.lastUsed, now) // Update LRU timestamp | 更新最后使用时间戳
		return e.client, nil
	}

	// 2) Cache miss: Create a new client | 缓存未命中：创建新的客户端
	c, err := createClient(config)
	if err != nil {
		return c, err // Fails fast if proxy string is invalid | 代理字符串异常时尽早报错拒绝
	}

	newEntry := &poolEntry{
		client:   c,
		lastUsed: now,
	}

	// 3) Concurrency Deduplication | 并发去重控制
	// Prevents identical clients from overwriting each other during concurrent cache misses.
	// 防止高并发下的缓存击穿导致多个相同配置的 Client 互相覆盖。
	actual, loaded := customPool.LoadOrStore(poolKey, newEntry)
	if loaded {
		// A concurrent goroutine already created and stored it. Clean up ours.
		// 其他并发协程抢先创建并存入了缓存，关闭我们刚创建的冗余底层连接。
		c.CloseIdleConnections()

		e := actual.(*poolEntry)
		atomic.StoreInt64(&e.lastUsed, now)
		return e.client, nil
	}

	// Successfully added a brand new client to the pool.
	// 真实新增了一个 Client 入池。
	atomic.AddInt64(&customPoolSize, 1)

	// 4) Non-blocking asynchronous maintenance | 惰性清理 (无阻塞设计)
	// Triggers GC for expired or over-capacity clients without blocking the current HTTP request.
	// 触发对过期或超量客户端的清理，绝不阻塞当前 HTTP 请求的生命周期。
	triggerPoolMaintenance()

	return c, nil
}

// triggerPoolMaintenance schedules the background garbage collection safely.
// 安全地调度后台清理任务。
func triggerPoolMaintenance() {
	// Mark that a maintenance job is required.
	// 标记“有清理需求”。
	atomic.StoreUint32(&maintenancePending, 1)

	// Lock-free pattern: Only one goroutine can transition from idle (0) to running (1).
	// 无锁 CAS 模式：无论瞬间涌入多少并发，永远只会有 1 个后台协程被唤醒去执行清理任务。
	if atomic.CompareAndSwapUint32(&maintenanceRunning, 0, 1) {
		go runPoolMaintenance()
	}
}

// runPoolMaintenance is the dedicated background worker for pool eviction.
// 专门执行连接池淘汰策略的后台工作协程。
func runPoolMaintenance() {
	for {
		// Consume the pending task flag.
		// 消费当前批次的清理需求。
		atomic.StoreUint32(&maintenancePending, 0)

		cleanupCustomPoolExpired() // TTL GC
		evictCustomPoolIfNeeded()  // LRU Eviction

		// Attempt to exit if no new tasks were triggered.
		// 没有新需求，尝试退出。
		if atomic.LoadUint32(&maintenancePending) == 0 {
			atomic.StoreUint32(&maintenanceRunning, 0)

			// Prevent "Lost Wakeup": If a new task arrived right before we set status to 0,
			// we reclaim the running status and continue.
			// 防止“丢失唤醒”：退出前若又来了新需求，我们通过 CAS 抢回执行权继续跑，确保任务不遗漏。
			if atomic.LoadUint32(&maintenancePending) == 1 &&
				atomic.CompareAndSwapUint32(&maintenanceRunning, 0, 1) {
				continue
			}
			return
		}
		// Continue next iteration if there are new pending tasks.
		// 发现新需求，继续下一轮循环。
	}
}

// cleanupCustomPoolExpired removes clients that haven't been used beyond the TTL.
// 清理超过 TTL 时间未被使用的闲置客户端。
func cleanupCustomPoolExpired() {
	// TryLock prevents CPU stampedes (惊群效应). If someone else is cleaning, just return.
	// 使用 TryLock，如果有其他协程正在扫地，直接下班，避免 CPU 阻塞。
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
					ae.client.CloseIdleConnections() // Release TCP sockets | 释放 TCP 句柄
				}
				atomic.AddInt64(&customPoolSize, -1)
			}
		}
		return true
	})
}

// evictCustomPoolIfNeeded enforces the MaxEntries limit using an LRU strategy.
// 超出容量上限时，使用 LRU (最近最少使用) 算法进行淘汰。
func evictCustomPoolIfNeeded() {
	// Early exit if within limits.
	// 提前判断，没超标直接退出。
	if atomic.LoadInt64(&customPoolSize) <= CustomPoolMaxEntries {
		return
	}

	// TryLock to avoid eviction backlogs.
	// 同样使用 TryLock 防止积压排队。
	if !poolEvictMu.TryLock() {
		return
	}
	defer poolEvictMu.Unlock()

	// Double check after acquiring lock.
	// 拿到锁后做 Double Check，防止误判。
	currentSize := atomic.LoadInt64(&customPoolSize)
	if currentSize <= CustomPoolMaxEntries {
		return
	}

	needRemove := int(currentSize - CustomPoolMaxEntries)

	// Collect all keys and timestamps for sorting (O(N) operation)
	// 收集所有 key 和时间戳准备排序
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

	// Sort by lastUsed ascending (oldest first)
	// 按照 lastUsed 升序排序 (最旧的排在前面)
	sort.Slice(items, func(i, j int) bool {
		return items[i].lastUsed < items[j].lastUsed
	})

	// Batch delete the oldest N elements. Avoids nested O(N) operations.
	// 批量干掉最旧的 N 个元素。这种做法将嵌套的时间复杂度降至 O(N log N)。
	for i := 0; i < needRemove && i < len(items); i++ {
		if actual, ok := customPool.LoadAndDelete(items[i].key); ok {
			if ae, ok := actual.(*poolEntry); ok && ae.client != nil {
				ae.client.CloseIdleConnections()
			}
			atomic.AddInt64(&customPoolSize, -1)
		}
	}
}

// Close gracefully shuts down all clients, releases TCP sockets and idle goroutines.
// Useful during application shutdown or hot-reloads.
//
// 优雅关闭所有客户端，释放空闲连接、goroutine 和端口。
// 在程序退出或微服务热更新时调用，防止 TIME_WAIT 积压。
func Close() {
	if httpclient != nil {
		httpclient.CloseIdleConnections()
	}

	// Acquire Lock: Blocks any new calls to GetDefineClient.
	// 申请写锁：在此期间，任何业务调用 GetDefineClient 创建新连接都会被阻塞挂起。
	poolMu.Lock()
	defer poolMu.Unlock()

	// Lock eviction to prevent background tasks from interfering.
	// 锁住淘汰逻辑，防止 Close 期间有异步协程过来打架。
	poolEvictMu.Lock()
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

	customPool.Clear()
}

// ================== Internal Helpers | 内部辅助函数 ==================

// createClient constructs the core http.Client and its underlying http.Transport.
// 核心组装函数，构造底层 Transport。
func createClient(config Config) (*http.Client, error) {
	transport := &http.Transport{
		// Force HTTP/2 protocol negotiation (Required for custom Transports in Go 1.13+)
		// 默认尝试 HTTP/2 (自定义 Transport 时必须显式设置为 true 才会启用 h2)
		ForceAttemptHTTP2: true,

		// Automatically read HTTP_PROXY/HTTPS_PROXY env variables by default
		// 默认使用环境变量代理
		Proxy: http.ProxyFromEnvironment,

		// Connection Pool Tuning | 连接池调优
		MaxIdleConns:        config.MaxIdleConns,        // Total idle pool size | 最大空闲连接总数
		MaxIdleConnsPerHost: config.MaxIdlePerHost,      // Idle pool per host | 单域名的最大空闲连接数
		MaxConnsPerHost:     config.MaxIdlePerHost << 1, // Hard limit on concurrency | 单域名的最大并发总数 (防下游雪崩)
		IdleConnTimeout:     90 * time.Second,           // Time before closing idle sockets | 空闲连接的存活时间

		// Basic Dialing parameters | 基础拨号与握手配置
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second, // TCP handshake timeout | TCP 建连超时
			KeepAlive: 30 * time.Second, // TCP Keep-Alive probe interval | TCP 探活周期
		}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second, // TLS handshake timeout | TLS 握手超时
		ExpectContinueTimeout: 1 * time.Second,  // 预期继续超时
	}

	// FirstBytesTimeout prevents dead proxies that establish TCP but hang indefinitely.
	// 首字节超时：专门对抗“假死”代理 (只建连不回数据)，极大地提高爬虫的鲁棒性。
	if config.FirstBytesTimeout > 0 {
		transport.ResponseHeaderTimeout = config.FirstBytesTimeout
	}

	// Override with custom proxy if provided.
	// 如果传入了自定义代理，则覆盖默认的环境变量代理。
	if config.Proxy != "" {
		if proxyURL, err := url.Parse(config.Proxy); err != nil {
			return nil, err // Fail fast on invalid proxy | 代理格式错误尽早拦截，防止真实 IP 泄露
		} else {
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	}

	// Skip HTTPS TLS verification (useful for packet sniffing proxies like Charles).
	// TLS 校验处理 (使用抓包工具时常用)。
	if config.SkipVerify {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	// Handle 'infinite timeout' trick. (-1 means no overall timeout in this package)
	// 处理无超时逻辑：若 config.Timeout 为 -1，则转为原生的 0 (无超时限制)。
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

// normalizeConfigForPool sanitizes edge case parameters to prevent cache fragmentation.
// 配置归一化：修复异常输入，防止产生无意义的冗余缓存 key。
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

// buildPoolKey creates a deterministic string key based on parameters that affect Transport behaviors.
// 构造缓存 Key。只有影响底层网络行为的参数才会参与 Key 生成，做到池化复用的最大化。
func buildPoolKey(c Config) string {
	proxyKey := c.Proxy
	if proxyKey == "" {
		proxyKey = "__ENV_PROXY__" // Distinguish from actual empty strings | 给予环境变量特殊标识，防止歧义
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

// normalizeProxyForKey unifies formatting of proxy strings to prevent duplicates (e.g., lowercase matching).
// 代理字符串归一化：消除大小写、尾部斜杠等差异导致的 Key 碎片化。
func normalizeProxyForKey(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	u, err := url.Parse(raw)
	if err != nil {
		return raw // Maintain original behavior for invalid urls | 如果解析失败保持原样，交给 createClient 去报错
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
