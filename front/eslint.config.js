import antfu from "@antfu/eslint-config"

// 更多自定义配置可查阅仓库：https://github.com/antfu/eslint-config
export default antfu(
  {
    // 使用外部格式化程序格式化 css、html、markdown 等文件
    formatters: true,
    // 启用样式规则
    stylistic: {
      // 缩进级别
      indent: 2,
      // 引号风格 'single' | 'double'
      quotes: "double",
      // 是否启用分号
      semi: false
    },
    // 忽略文件
    ignores: []
  },
  {
    // 对所有文件都生效的规则
    rules: {
      // vue
      "vue/block-order": ["error", { order: ["script", "template", "style"] }],
      "vue/attributes-order": "off",
      // 关闭自闭合与内容换行相关（减少无谓告警）
      "vue/html-self-closing": "off",
      "vue/singleline-html-element-content-newline": "off",
      "vue/multiline-html-element-content-newline": "off",

      // ts
      "ts/no-use-before-define": "off",
      // 关闭类型导入强制（允许行内 type 指定）
      "ts/consistent-type-imports": "off",

      // import 顺序（关闭强制排序）
      "import/order": "off",

      // node
      "node/prefer-global/process": "off",

      // style
      // 关闭大括号样式强制，避免“花括号换行”噪音
      "style/brace-style": "off",
      // 允许同一行多个语句（减少噪音）
      "max-statements-per-line": "off",
      "style/comma-dangle": ["error", "never"],

      // regexp
      "regexp/no-unused-capturing-group": "off",

      // other
      "no-console": "off",
      "no-debugger": "off",
      "symbol-description": "off",
      "antfu/if-newline": "off"
    }
  }
)
