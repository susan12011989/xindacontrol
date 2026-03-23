package gostapi

// Response API 响应结构
type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

// Config GOST 配置
type Config struct {
	Services   []ServiceConfig   `json:"services,omitempty"`
	Chains     []ChainConfig     `json:"chains,omitempty"`
	Hops       []HopConfig       `json:"hops,omitempty"`
	Authers    []AutherConfig    `json:"authers,omitempty"`
	Admissions []AdmissionConfig `json:"admissions,omitempty"`
	Bypasses   []BypassConfig    `json:"bypasses,omitempty"`
	Resolvers  []ResolverConfig  `json:"resolvers,omitempty"`
	Hosts      []HostsConfig     `json:"hosts,omitempty"`
	Ingresses  []IngressConfig   `json:"ingresses,omitempty"`
	Limiters   []LimiterConfig   `json:"limiters,omitempty"`
	CLimiters  []LimiterConfig   `json:"climiters,omitempty"`
	RLimiters  []LimiterConfig   `json:"rlimiters,omitempty"`
	Recorders  []RecorderConfig  `json:"recorders,omitempty"`
	Observers  []ObserverConfig  `json:"observers,omitempty"`
	Loggers    []LoggerConfig    `json:"loggers,omitempty"`
	Routers    []RouterConfig    `json:"routers,omitempty"`
	SDs        []SDConfig        `json:"sds,omitempty"`
	API        *APIConfig        `json:"api,omitempty"`
	Metrics    *MetricsConfig    `json:"metrics,omitempty"`
	Log        *LogConfig        `json:"log,omitempty"`
	Profiling  *ProfilingConfig  `json:"profiling,omitempty"`
	TLS        *TLSConfig        `json:"tls,omitempty"`
}

// ServiceList 服务列表
type ServiceList struct {
	Count int             `json:"count"`
	List  []ServiceConfig `json:"list"`
}

// ChainList 链列表
type ChainList struct {
	Count int           `json:"count"`
	List  []ChainConfig `json:"list"`
}

// ServiceConfig 服务配置
type ServiceConfig struct {
	Name       string           `json:"name"`
	Addr       string           `json:"addr"`
	Interface  string           `json:"interface,omitempty"`
	Handler    *HandlerConfig   `json:"handler,omitempty"`
	Listener   *ListenerConfig  `json:"listener,omitempty"`
	Forwarder  *ForwarderConfig `json:"forwarder,omitempty"`
	Admission  string           `json:"admission,omitempty"`
	Admissions []string         `json:"admissions,omitempty"`
	Bypass     string           `json:"bypass,omitempty"`
	Bypasses   []string         `json:"bypasses,omitempty"`
	Resolver   string           `json:"resolver,omitempty"`
	Hosts      string           `json:"hosts,omitempty"`
	Limiter    string           `json:"limiter,omitempty"`
	CLimiter   string           `json:"climiter,omitempty"`
	RLimiter   string           `json:"rlimiter,omitempty"`
	Observer   string           `json:"observer,omitempty"`
	Logger     string           `json:"logger,omitempty"`
	Loggers    []string         `json:"loggers,omitempty"`
	Recorders  []RecorderObject `json:"recorders,omitempty"`
	Metadata   map[string]any   `json:"metadata,omitempty"`
	SockOpts   *SockOptsConfig  `json:"sockopts,omitempty"`
	Status     *ServiceStatus   `json:"status,omitempty"`
}

// HandlerConfig 处理器配置
type HandlerConfig struct {
	Type       string            `json:"type"`
	Auth       *AuthConfig       `json:"auth,omitempty"`
	Auther     string            `json:"auther,omitempty"`
	Authers    []string          `json:"authers,omitempty"`
	TLS        *TLSConfig        `json:"tls,omitempty"`
	Chain      string            `json:"chain,omitempty"`
	ChainGroup *ChainGroupConfig `json:"chainGroup,omitempty"`
	Retries    int               `json:"retries,omitempty"`
	Observer   string            `json:"observer,omitempty"`
	Limiter    string            `json:"limiter,omitempty"`
	Metadata   map[string]any    `json:"metadata,omitempty"`
}

// ListenerConfig 监听器配置
type ListenerConfig struct {
	Type       string            `json:"type"`
	Auth       *AuthConfig       `json:"auth,omitempty"`
	Auther     string            `json:"auther,omitempty"`
	Authers    []string          `json:"authers,omitempty"`
	TLS        *TLSConfig        `json:"tls,omitempty"`
	Chain      string            `json:"chain,omitempty"`
	ChainGroup *ChainGroupConfig `json:"chainGroup,omitempty"`
	Metadata   map[string]any    `json:"metadata,omitempty"`
}

// AuthConfig 认证配置
type AuthConfig struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	File     string `json:"file,omitempty"`
}

// TLSConfig TLS 配置
type TLSConfig struct {
	CertFile     string      `json:"certFile,omitempty"`
	KeyFile      string      `json:"keyFile,omitempty"`
	CAFile       string      `json:"caFile,omitempty"`
	Secure       bool        `json:"secure,omitempty"`
	ServerName   string      `json:"serverName,omitempty"`
	Options      *TLSOptions `json:"options,omitempty"`
	CommonName   string      `json:"commonName,omitempty"`
	Organization string      `json:"organization,omitempty"`
	Validity     *Duration   `json:"validity,omitempty"`
}

// TLSOptions TLS 选项
type TLSOptions struct {
	MinVersion   string   `json:"minVersion,omitempty"`
	MaxVersion   string   `json:"maxVersion,omitempty"`
	CipherSuites []string `json:"cipherSuites,omitempty"`
	ALPN         []string `json:"alpn,omitempty"`
}

// ChainConfig 链配置
type ChainConfig struct {
	Name     string         `json:"name"`
	Hops     []HopConfig    `json:"hops,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// ChainGroupConfig 链组配置
type ChainGroupConfig struct {
	Chains   []string        `json:"chains,omitempty"`
	Selector *SelectorConfig `json:"selector,omitempty"`
}

// HopConfig 跃点配置
type HopConfig struct {
	Name      string          `json:"name"`
	Nodes     []NodeConfig    `json:"nodes,omitempty"`
	Selector  *SelectorConfig `json:"selector,omitempty"`
	Bypass    string          `json:"bypass,omitempty"`
	Bypasses  []string        `json:"bypasses,omitempty"`
	Resolver  string          `json:"resolver,omitempty"`
	Hosts     string          `json:"hosts,omitempty"`
	Interface string          `json:"interface,omitempty"`
	SockOpts  *SockOptsConfig `json:"sockopts,omitempty"`
	Metadata  map[string]any  `json:"metadata,omitempty"`
	File      *FileLoader     `json:"file,omitempty"`
	HTTP      *HTTPLoader     `json:"http,omitempty"`
	Redis     *RedisLoader    `json:"redis,omitempty"`
	Plugin    *PluginConfig   `json:"plugin,omitempty"`
	Reload    *Duration       `json:"reload,omitempty"`
}

// NodeConfig 节点配置
type NodeConfig struct {
	Name      string             `json:"name,omitempty"`
	Addr      string             `json:"addr"`
	Network   string             `json:"network,omitempty"`
	Connector *ConnectorConfig   `json:"connector,omitempty"`
	Dialer    *DialerConfig      `json:"dialer,omitempty"`
	Bypass    string             `json:"bypass,omitempty"`
	Bypasses  []string           `json:"bypasses,omitempty"`
	Resolver  string             `json:"resolver,omitempty"`
	Hosts     string             `json:"hosts,omitempty"`
	Interface string             `json:"interface,omitempty"`
	SockOpts  *SockOptsConfig    `json:"sockopts,omitempty"`
	Metadata  map[string]any     `json:"metadata,omitempty"`
	HTTP      *HTTPNodeConfig    `json:"http,omitempty"`
	TLS       *TLSNodeConfig     `json:"tls,omitempty"`
	Filter    *NodeFilterConfig  `json:"filter,omitempty"`
	Matcher   *NodeMatcherConfig `json:"matcher,omitempty"`
	Netns     string             `json:"netns,omitempty"`
}

// ConnectorConfig 连接器配置
type ConnectorConfig struct {
	Type     string         `json:"type"`
	Auth     *AuthConfig    `json:"auth,omitempty"`
	TLS      *TLSConfig     `json:"tls,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// DialerConfig 拨号器配置
type DialerConfig struct {
	Type     string         `json:"type"`
	Auth     *AuthConfig    `json:"auth,omitempty"`
	TLS      *TLSConfig     `json:"tls,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// ForwarderConfig 转发器配置
type ForwarderConfig struct {
	Nodes    []ForwardNodeConfig `json:"nodes,omitempty"`
	Selector *SelectorConfig     `json:"selector,omitempty"`
	Hop      string              `json:"hop,omitempty"`
	Name     string              `json:"name,omitempty"` // Deprecated
}

// ForwardNodeConfig 转发节点配置
type ForwardNodeConfig struct {
	Name     string             `json:"name,omitempty"`
	Addr     string             `json:"addr"`
	Network  string             `json:"network,omitempty"`
	Protocol string             `json:"protocol,omitempty"` // Deprecated
	Host     string             `json:"host,omitempty"`     // Deprecated
	Path     string             `json:"path,omitempty"`     // Deprecated
	Auth     *AuthConfig        `json:"auth,omitempty"`
	Bypass   string             `json:"bypass,omitempty"`
	Bypasses []string           `json:"bypasses,omitempty"`
	HTTP     *HTTPNodeConfig    `json:"http,omitempty"`
	TLS      *TLSNodeConfig     `json:"tls,omitempty"`
	Filter   *NodeFilterConfig  `json:"filter,omitempty"`
	Matcher  *NodeMatcherConfig `json:"matcher,omitempty"`
	Metadata map[string]any     `json:"metadata,omitempty"`
}

// SelectorConfig 选择器配置
type SelectorConfig struct {
	Strategy    string    `json:"strategy,omitempty"`
	MaxFails    int       `json:"maxFails,omitempty"`
	FailTimeout *Duration `json:"failTimeout,omitempty"`
}

// SockOptsConfig Socket 选项配置
type SockOptsConfig struct {
	Mark int `json:"mark,omitempty"`
}

// ServiceStatus 服务状态
type ServiceStatus struct {
	State      string         `json:"state,omitempty"`
	CreateTime int64          `json:"createTime,omitempty"`
	Events     []ServiceEvent `json:"events,omitempty"`
	Stats      *ServiceStats  `json:"stats,omitempty"`
}

// ServiceEvent 服务事件
type ServiceEvent struct {
	Msg  string `json:"msg"`
	Time int64  `json:"time"`
}

// ServiceStats 服务统计
type ServiceStats struct {
	TotalConns   uint64 `json:"totalConns"`
	CurrentConns uint64 `json:"currentConns"`
	InputBytes   uint64 `json:"inputBytes"`
	OutputBytes  uint64 `json:"outputBytes"`
	TotalErrs    uint64 `json:"totalErrs"`
}

// RecorderObject 记录器对象
type RecorderObject struct {
	Name     string         `json:"name"`
	Record   string         `json:"record"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// HTTPNodeConfig HTTP 节点配置
type HTTPNodeConfig struct {
	Host           string                  `json:"host,omitempty"`
	Header         map[string]string       `json:"header,omitempty"` // Deprecated
	RequestHeader  map[string]string       `json:"requestHeader,omitempty"`
	ResponseHeader map[string]string       `json:"responseHeader,omitempty"`
	Auth           *AuthConfig             `json:"auth,omitempty"`
	Rewrite        []HTTPURLRewriteConfig  `json:"rewrite,omitempty"` // Deprecated
	RewriteURL     []HTTPURLRewriteConfig  `json:"rewriteURL,omitempty"`
	RewriteBody    []HTTPBodyRewriteConfig `json:"rewriteBody,omitempty"`
}

// HTTPURLRewriteConfig HTTP URL 重写配置
type HTTPURLRewriteConfig struct {
	Match       string `json:"Match"`
	Replacement string `json:"Replacement"`
}

// HTTPBodyRewriteConfig HTTP Body 重写配置
type HTTPBodyRewriteConfig struct {
	Type        string `json:"Type,omitempty"`
	Match       string `json:"Match"`
	Replacement string `json:"Replacement"`
}

// TLSNodeConfig TLS 节点配置
type TLSNodeConfig struct {
	Secure     bool        `json:"secure,omitempty"`
	ServerName string      `json:"serverName,omitempty"`
	Options    *TLSOptions `json:"options,omitempty"`
}

// NodeFilterConfig 节点过滤器配置
type NodeFilterConfig struct {
	Protocol string `json:"protocol,omitempty"`
	Host     string `json:"host,omitempty"`
	Path     string `json:"path,omitempty"`
}

// NodeMatcherConfig 节点匹配器配置
type NodeMatcherConfig struct {
	Rule     string `json:"rule,omitempty"`
	Priority int    `json:"priority,omitempty"`
}

// AutherConfig 认证器配置
type AutherConfig struct {
	Name   string        `json:"name"`
	Auths  []AuthConfig  `json:"auths,omitempty"`
	File   *FileLoader   `json:"file,omitempty"`
	HTTP   *HTTPLoader   `json:"http,omitempty"`
	Redis  *RedisLoader  `json:"redis,omitempty"`
	Plugin *PluginConfig `json:"plugin,omitempty"`
	Reload *Duration     `json:"reload,omitempty"`
}

// AdmissionConfig 准入控制配置
type AdmissionConfig struct {
	Name      string        `json:"name"`
	Matchers  []string      `json:"matchers,omitempty"`
	Whitelist bool          `json:"whitelist,omitempty"`
	Reverse   bool          `json:"reverse,omitempty"` // Deprecated
	File      *FileLoader   `json:"file,omitempty"`
	HTTP      *HTTPLoader   `json:"http,omitempty"`
	Redis     *RedisLoader  `json:"redis,omitempty"`
	Plugin    *PluginConfig `json:"plugin,omitempty"`
	Reload    *Duration     `json:"reload,omitempty"`
}

// BypassConfig 绕过配置
type BypassConfig struct {
	Name      string        `json:"name"`
	Matchers  []string      `json:"matchers,omitempty"`
	Whitelist bool          `json:"whitelist,omitempty"`
	Reverse   bool          `json:"reverse,omitempty"` // Deprecated
	File      *FileLoader   `json:"file,omitempty"`
	HTTP      *HTTPLoader   `json:"http,omitempty"`
	Redis     *RedisLoader  `json:"redis,omitempty"`
	Plugin    *PluginConfig `json:"plugin,omitempty"`
	Reload    *Duration     `json:"reload,omitempty"`
}

// ResolverConfig 解析器配置
type ResolverConfig struct {
	Name        string             `json:"name"`
	Nameservers []NameserverConfig `json:"nameservers,omitempty"`
	Plugin      *PluginConfig      `json:"plugin,omitempty"`
}

// NameserverConfig 名称服务器配置
type NameserverConfig struct {
	Addr     string    `json:"addr"`
	Chain    string    `json:"chain,omitempty"`
	Prefer   string    `json:"prefer,omitempty"`
	ClientIP string    `json:"clientIP,omitempty"`
	Hostname string    `json:"hostname,omitempty"`
	Only     string    `json:"only,omitempty"`
	Async    bool      `json:"async,omitempty"`
	TTL      *Duration `json:"ttl,omitempty"`
	Timeout  *Duration `json:"timeout,omitempty"`
}

// HostsConfig 主机配置
type HostsConfig struct {
	Name     string              `json:"name"`
	Mappings []HostMappingConfig `json:"mappings,omitempty"`
	File     *FileLoader         `json:"file,omitempty"`
	HTTP     *HTTPLoader         `json:"http,omitempty"`
	Redis    *RedisLoader        `json:"redis,omitempty"`
	Plugin   *PluginConfig       `json:"plugin,omitempty"`
	Reload   *Duration           `json:"reload,omitempty"`
}

// HostMappingConfig 主机映射配置
type HostMappingConfig struct {
	IP       string   `json:"ip"`
	Hostname string   `json:"hostname"`
	Aliases  []string `json:"aliases,omitempty"`
}

// IngressConfig 入口配置
type IngressConfig struct {
	Name   string              `json:"name"`
	Rules  []IngressRuleConfig `json:"rules,omitempty"`
	File   *FileLoader         `json:"file,omitempty"`
	HTTP   *HTTPLoader         `json:"http,omitempty"`
	Redis  *RedisLoader        `json:"redis,omitempty"`
	Plugin *PluginConfig       `json:"plugin,omitempty"`
	Reload *Duration           `json:"reload,omitempty"`
}

// IngressRuleConfig 入口规则配置
type IngressRuleConfig struct {
	Hostname string `json:"hostname"`
	Endpoint string `json:"endpoint"`
}

// LimiterConfig 限制器配置
type LimiterConfig struct {
	Name   string        `json:"name"`
	Limits []string      `json:"limits,omitempty"`
	File   *FileLoader   `json:"file,omitempty"`
	HTTP   *HTTPLoader   `json:"http,omitempty"`
	Redis  *RedisLoader  `json:"redis,omitempty"`
	Plugin *PluginConfig `json:"plugin,omitempty"`
	Reload *Duration     `json:"reload,omitempty"`
}

// RecorderConfig 记录器配置
type RecorderConfig struct {
	Name   string         `json:"name"`
	File   *FileRecorder  `json:"file,omitempty"`
	HTTP   *HTTPRecorder  `json:"http,omitempty"`
	TCP    *TCPRecorder   `json:"tcp,omitempty"`
	Redis  *RedisRecorder `json:"redis,omitempty"`
	Plugin *PluginConfig  `json:"plugin,omitempty"`
}

// FileRecorder 文件记录器
type FileRecorder struct {
	Path     string             `json:"path"`
	Sep      string             `json:"sep,omitempty"`
	Rotation *LogRotationConfig `json:"rotation,omitempty"`
}

// HTTPRecorder HTTP 记录器
type HTTPRecorder struct {
	URL     string            `json:"url"`
	Header  map[string]string `json:"header,omitempty"`
	Timeout *Duration         `json:"timeout,omitempty"`
}

// TCPRecorder TCP 记录器
type TCPRecorder struct {
	Addr    string    `json:"addr"`
	Timeout *Duration `json:"timeout,omitempty"`
}

// RedisRecorder Redis 记录器
type RedisRecorder struct {
	Addr     string `json:"addr"`
	DB       int    `json:"db,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Key      string `json:"key,omitempty"`
	Type     string `json:"type,omitempty"`
}

// ObserverConfig 观察器配置
type ObserverConfig struct {
	Name   string        `json:"name"`
	Plugin *PluginConfig `json:"plugin,omitempty"`
}

// LoggerConfig 日志器配置
type LoggerConfig struct {
	Name string     `json:"name"`
	Log  *LogConfig `json:"log,omitempty"`
}

// RouterConfig 路由器配置
type RouterConfig struct {
	Name   string              `json:"name"`
	Routes []RouterRouteConfig `json:"routes,omitempty"`
	File   *FileLoader         `json:"file,omitempty"`
	HTTP   *HTTPLoader         `json:"http,omitempty"`
	Redis  *RedisLoader        `json:"redis,omitempty"`
	Plugin *PluginConfig       `json:"plugin,omitempty"`
	Reload *Duration           `json:"reload,omitempty"`
}

// RouterRouteConfig 路由器路由配置
type RouterRouteConfig struct {
	Net     string `json:"net,omitempty"` // Deprecated
	Dst     string `json:"dst,omitempty"`
	Gateway string `json:"gateway,omitempty"`
}

// SDConfig 服务发现配置
type SDConfig struct {
	Name   string        `json:"name"`
	Plugin *PluginConfig `json:"plugin,omitempty"`
}

// APIConfig API 配置
type APIConfig struct {
	Addr       string      `json:"addr"`
	PathPrefix string      `json:"pathPrefix,omitempty"`
	AccessLog  bool        `json:"accesslog,omitempty"`
	Auth       *AuthConfig `json:"auth,omitempty"`
	Auther     string      `json:"auther,omitempty"`
}

// MetricsConfig 指标配置
type MetricsConfig struct {
	Addr   string      `json:"addr"`
	Path   string      `json:"path,omitempty"`
	Auth   *AuthConfig `json:"auth,omitempty"`
	Auther string      `json:"auther,omitempty"`
}

// LogConfig 日志配置
type LogConfig struct {
	Output   string             `json:"output,omitempty"`
	Level    string             `json:"level,omitempty"`
	Format   string             `json:"format,omitempty"`
	Rotation *LogRotationConfig `json:"rotation,omitempty"`
}

// LogRotationConfig 日志轮转配置
type LogRotationConfig struct {
	MaxSize    int  `json:"maxSize,omitempty"`
	MaxAge     int  `json:"maxAge,omitempty"`
	MaxBackups int  `json:"maxBackups,omitempty"`
	LocalTime  bool `json:"localTime,omitempty"`
	Compress   bool `json:"compress,omitempty"`
}

// ProfilingConfig 性能分析配置
type ProfilingConfig struct {
	Addr string `json:"addr"`
}

// FileLoader 文件加载器
type FileLoader struct {
	Path string `json:"path"`
}

// HTTPLoader HTTP 加载器
type HTTPLoader struct {
	URL     string    `json:"url"`
	Timeout *Duration `json:"timeout,omitempty"`
}

// RedisLoader Redis 加载器
type RedisLoader struct {
	Addr     string `json:"addr"`
	DB       int    `json:"db,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Key      string `json:"key,omitempty"`
	Type     string `json:"type,omitempty"`
}

// PluginConfig 插件配置
type PluginConfig struct {
	Type    string     `json:"type,omitempty"`
	Addr    string     `json:"addr"`
	TLS     *TLSConfig `json:"tls,omitempty"`
	Token   string     `json:"token,omitempty"`
	Timeout *Duration  `json:"timeout,omitempty"`
}

// Duration 时间间隔（纳秒）
type Duration int64

// DurationSeconds 创建秒级 Duration 指针
func DurationSeconds(seconds int) *Duration {
	d := Duration(int64(seconds) * 1_000_000_000)
	return &d
}
