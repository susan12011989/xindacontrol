// 运行时配置（可被 Docker 环境变量替换）
// 此文件在运行时加载，允许动态配置 API 地址等参数
window.__RUNTIME_CONFIG__ = {
  // API 基础路径（相对路径表示同域，绝对路径表示跨域）
  API_BASE_URL: '/server/v1',

  // WebSocket 地址（可选，留空使用默认）
  WS_URL: '',

  // 其他运行时配置
  APP_TITLE: 'Control Server'
};
