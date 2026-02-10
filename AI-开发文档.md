# Teamgram Enterprise Control AI 开发文档

> 本文档旨在帮助 AI 快速上手开发后台管理系统的前后端功能。

---

## 📋 目录

- [1. 项目概述](#1-项目概述)
- [2. 前端开发指南](#2-前端开发指南)
- [3. 后端开发指南](#3-后端开发指南)
- [4. 完整开发流程示例](#4-完整开发流程示例)
- [5. 常见问题与最佳实践](#5-常见问题与最佳实践)

---

## 1. 项目概述

### 1.1 项目结构

```
control/
├── front/          # 前端项目（Vue3 + TypeScript + Element Plus）
│   ├── src/
│   │   ├── pages/          # 页面组件
│   │   │   └── demo/       # ⭐ 参考示例（必看）
│   │   ├── common/         # 通用模块（@@别名）
│   │   │   ├── apis/       # API 接口定义
│   │   │   ├── composables/# 可复用组合式函数
│   │   │   ├── components/ # 通用组件
│   │   │   └── utils/      # 工具函数
│   │   ├── pinia/          # 状态管理
│   │   ├── router/         # 路由配置
│   │   └── http/           # HTTP 请求封装
│   └── vite.config.ts      # Vite 配置
└── server/         # 后端项目（Go + Gin + Xorm）
    ├── internal/server/    # 管理后台模块
    │   ├── model/          # 数据模型（请求/响应结构体）
    │   ├── router/         # 路由定义（按模块分目录）
    │   │   ├── auth/       # 认证路由
    │   │   ├── merchant/   # 商户路由
    │   │   └── cloud_aliyun/ # 阿里云路由
    │   ├── service/        # 业务逻辑（按模块分目录）
    │   │   ├── auth/       # 认证服务
    │   │   ├── merchant/   # 商户服务
    │   │   └── cloud_aliyun/ # 阿里云服务
    │   ├── middleware/     # 中间件
    │   ├── cloud/          # 云服务SDK封装
    │   │   └── aliyun/     # 阿里云SDK
    │   ├── cfg/            # 配置
    │   └── static/         # 静态文件（前端构建产物）
    └── pkg/                # 公共包
        ├── dbs/            # 数据库连接
        ├── result/         # 统一响应格式
        ├── token_manager/  # Token 管理
        └── entity/         # 数据库实体
```

---

## 2. 前端开发指南

### 2.1 技术栈

- **框架**: Vue 3.5.13 + TypeScript
- **UI 库**: Element Plus 2.9.5
- **表格**: vxe-table 4.6.25 (用于复杂表格)
- **状态管理**: Pinia 3.0.1
- **路由**: Vue Router 4.5.0
- **HTTP**: Axios 1.7.9
- **构建**: Vite 6.1.1
- **CSS**: UnoCSS + SCSS

### 2.2 目录结构说明

#### 2.2.1 路径别名

```typescript
@   -> src/          // 根目录
@@  -> src/common/   // 通用模块目录
```

#### 2.2.2 重要目录

- `src/pages/demo/` - **必看示例目录**，包含标准写法参考
- `src/common/apis/` - API 接口定义（按模块分类）
- `src/common/composables/` - 可复用的组合式函数
- `src/common/components/` - 全局通用组件

### 2.3 开发规范（参考 demo）

#### 2.3.1 页面组件标准写法

**参考**: `front/src/pages/demo/vxe-table/index.vue`

```vue
<script lang="ts" setup>
// 1. 导入类型定义（放在最前面）
import type { TableResponseData } from "@@/apis/tables/type"
import type { VxeGridInstance, VxeGridProps } from "vxe-table"

// 2. 导入 API 函数
import { getTableDataApi, deleteTableDataApi } from "@@/apis/tables"

// 3. 定义组件名称（必须）
defineOptions({
  name: "YourComponentName"  // 必须设置唯一的组件名
})

// 4. 定义数据类型
interface RowMeta {
  id: number
  username: string
  // ... 其他字段
}

// 5. 定义响应式数据
const xGridDom = ref<VxeGridInstance>()
const loading = ref(false)

// 6. 使用 reactive 定义配置对象
const xGridOpt: VxeGridProps = reactive({
  loading: true,
  autoResize: true,
  pagerConfig: { align: "right" },
  // ... 其他配置
})

// 7. 定义业务逻辑（CRUD 操作）
const crudStore = reactive({
  commitQuery: () => xGridDom.value?.commitProxy("query"),
  onShowModal: (row?: RowMeta) => {
    // 显示弹窗逻辑
  },
  onDelete: (row: RowMeta) => {
    // 删除逻辑
  }
})
</script>

<template>
  <div class="app-container">
    <!-- 页面内容 -->
  </div>
</template>

<style lang="scss" scoped>
/* 组件样式 */
</style>
```

#### 2.3.2 API 接口定义

**位置**: `src/common/apis/{模块名}/`

**文件结构**:
```
apis/users/
├── index.ts    # API 函数定义
└── type.ts     # TypeScript 类型定义
```

**示例**: `front/src/common/apis/users/index.ts`

```typescript
import type * as Users from "./type"
import { request } from "@/http/axios"

/** 获取用户列表 */
export function getUserList(params: Users.QueryUsersReq) {
  return request<Users.QueryUsersResponseData>({
    url: "users",
    method: "get",
    params
  })
}

/** 更新用户 */
export function updateUser(id: number, data: Users.UpdateUserReq) {
  return request({
    url: `users/${id}`,
    method: "put",
    data
  })
}

/** 删除用户 */
export function deleteUser(id: number) {
  return request({
    url: `users/${id}`,
    method: "delete"
  })
}
```

**类型定义**: `front/src/common/apis/users/type.ts`

```typescript
import type { ApiResponseData } from "@/common/apis/type"

// 基础分页类型
export interface Pagination {
  page: number
  size: number
}

// 查询请求
export interface QueryUsersReq extends Pagination {
  id?: number
  username?: string
  phone?: string
}

// 更新请求
export interface UpdateUserReq {
  first_name?: string
  last_name?: string
  username?: string
}

// 响应数据
export interface UserResp {
  id: number
  username: string
  phone: string
  created_at: string
}

// 列表响应
export interface QueryUsersResponse {
  list: UserResp[]
  total: number
}

// API 响应类型（包装）
export type QueryUsersResponseData = ApiResponseData<QueryUsersResponse>
export type UserDetailResponseData = ApiResponseData<UserResp>
```

#### 2.3.3 使用 Composable 函数

**参考**: `front/src/pages/demo/composable-demo/use-fetch-select.vue`

**场景 1: 使用现有 Composable**

```vue
<script lang="ts" setup>
import { useFetchSelect } from "@@/composables/useFetchSelect"
import { getSelectDataApi } from "./apis/use-fetch-select"

// 直接使用现成的 Composable
const { loading, options, value } = useFetchSelect({
  api: getSelectDataApi
})
</script>

<template>
  <el-select v-model="value" :loading="loading">
    <el-option v-for="item in options" :key="item.value" v-bind="item" />
  </el-select>
</template>
```

**场景 2: 使用分页 Composable**

```vue
<script lang="ts" setup>
import { usePagination } from "@@/composables/usePagination"

const { paginationData, handleCurrentChange, handleSizeChange } = usePagination({
  pageSize: 20  // 可选的初始配置
})

// 监听分页变化，重新加载数据
watch([
  () => paginationData.currentPage,
  () => paginationData.pageSize
], () => {
  loadData()
})
</script>

<template>
  <el-pagination
    v-model:current-page="paginationData.currentPage"
    v-model:page-size="paginationData.pageSize"
    :page-sizes="paginationData.pageSizes"
    :layout="paginationData.layout"
    :total="paginationData.total"
    @size-change="handleSizeChange"
    @current-change="handleCurrentChange"
  />
</template>
```

**常用 Composables**:

- `usePagination` - 分页逻辑
- `useFetchSelect` - 下拉选择器数据获取
- `useFullscreenLoading` - 全屏加载
- `useWatermark` - 水印

#### 2.3.4 VXE Table 使用规范

**参考**: `front/src/pages/demo/vxe-table/index.vue`

**适用场景**: 复杂表格、需要高级功能（虚拟滚动、树形数据、复杂表单等）

```vue
<script lang="ts" setup>
import type { VxeGridInstance, VxeGridProps } from "vxe-table"

const xGridDom = ref<VxeGridInstance>()
const xGridOpt: VxeGridProps = reactive({
  loading: true,
  autoResize: true,
  // 分页配置
  pagerConfig: {
    align: "right"
  },
  // 表单配置（搜索栏）
  formConfig: {
    items: [
      {
        field: "username",
        itemRender: {
          name: "$input",
          props: { placeholder: "用户名", clearable: true }
        }
      }
    ]
  },
  // 工具栏配置
  toolbarConfig: {
    refresh: true,
    custom: true,
    slots: { buttons: "toolbar-btns" }
  },
  // 列配置
  columns: [
    { type: "checkbox", width: "50px" },
    { field: "username", title: "用户名" },
    { field: "phone", title: "手机号" },
    {
      title: "操作",
      width: "150px",
      fixed: "right",
      slots: { default: "row-operate" }
    }
  ],
  // 数据代理配置
  proxyConfig: {
    seq: true,
    form: true,
    autoLoad: true,
    props: { total: "total" },
    ajax: {
      query: ({ page, form }) => {
        return new Promise((resolve) => {
          const params = {
            username: form.username || "",
            size: page.pageSize,
            currentPage: page.currentPage
          }
          getTableDataApi(params).then((res) => {
            resolve({
              total: res.data.total,
              result: res.data.list
            })
          })
        })
      }
    }
  }
})

// CRUD 操作
const crudStore = reactive({
  commitQuery: () => xGridDom.value?.commitProxy("query"),
  onDelete: (row) => {
    ElMessageBox.confirm(`确定删除 ${row.username}?`, "提示").then(() => {
      deleteApi(row.id).then(() => {
        ElMessage.success("删除成功")
        crudStore.commitQuery()
      })
    })
  }
})
</script>

<template>
  <div class="app-container">
    <vxe-grid ref="xGridDom" v-bind="xGridOpt">
      <!-- 工具栏按钮 -->
      <template #toolbar-btns>
        <vxe-button status="primary" @click="crudStore.onShowModal()">
          新增
        </vxe-button>
      </template>
      <!-- 操作列 -->
      <template #row-operate="{ row }">
        <el-button link type="primary" @click="crudStore.onShowModal(row)">
          修改
        </el-button>
        <el-button link type="danger" @click="crudStore.onDelete(row)">
          删除
        </el-button>
      </template>
    </vxe-grid>
  </div>
</template>
```

#### 2.3.5 路由配置

**位置**: `front/src/router/index.ts`

```typescript
import type { RouteRecordRaw } from "vue-router"

const Layouts = () => import("@/layouts/index.vue")

// 常驻路由
export const constantRoutes: RouteRecordRaw[] = [
  {
    path: "/",
    component: Layouts,
    redirect: "/dashboard",
    children: [
      {
        path: "dashboard",
        component: () => import("@/pages/dashboard/index.vue"),
        name: "Dashboard",
        meta: {
          title: "首页",
          svgIcon: "dashboard",
          affix: true  // 固定标签
        }
      }
    ]
  }
]

// 动态路由（需要权限）
export const dynamicRoutes: RouteRecordRaw[] = [
  {
    path: "/users",
    component: Layouts,
    redirect: "/users/list",
    name: "Users",
    meta: {
      title: "用户管理",
      elIcon: "User",
      roles: ["admin"],  // 角色权限
      alwaysShow: true   // 始终显示根菜单
    },
    children: [
      {
        path: "list",
        component: () => import("@/pages/users/list.vue"),
        name: "UsersList",
        meta: {
          title: "用户列表",
          roles: ["admin"]
        }
      }
    ]
  }
]
```

#### 2.3.6 状态管理 (Pinia)

**参考**: `front/src/pinia/stores/user.ts`

```typescript
import { defineStore } from "pinia"
import { getCurrentUserApi } from "@@/apis/users"
import { getToken, setToken, removeToken } from "@@/utils/cache/cookies"

export const useUserStore = defineStore("user", () => {
  const token = ref<string>(getToken() || "")
  const username = ref<string>("")
  const roles = ref<string[]>([])

  // 设置 Token
  const setTokenValue = (value: string) => {
    setToken(value)
    token.value = value
  }

  // 获取用户信息
  const getInfo = async () => {
    const { data } = await getCurrentUserApi()
    username.value = data.username
    roles.value = data.roles || []
  }

  // 登出
  const logout = () => {
    removeToken()
    token.value = ""
    roles.value = []
  }

  return { token, username, roles, setTokenValue, getInfo, logout }
})
```

### 2.4 样式规范

#### 2.4.1 使用 SCSS

```scss
<style lang="scss" scoped>
.app-container {
  padding: 20px;
  
  .section-title {
    font-size: 18px;
    font-weight: 600;
    margin-bottom: 16px;
    padding-left: 12px;
    border-left: 4px solid var(--el-color-primary);
  }
  
  .stat-card {
    &:hover {
      transform: translateY(-4px);
    }
  }
}

// 响应式设计
@media (max-width: 768px) {
  .app-container {
    padding: 12px;
  }
}
</style>
```

#### 2.4.2 使用 UnoCSS（原子化 CSS）

```vue
<template>
  <!-- 
    常用类名:
    - flex, items-center, justify-between
    - p-4, m-2, gap-4
    - text-lg, font-bold
    - bg-blue-500, text-white
  -->
  <div class="flex items-center justify-between p-4 gap-2">
    <span class="text-lg font-bold">标题</span>
  </div>
</template>
```

### 2.5 常用工具函数

#### 2.5.1 日期格式化

```typescript
import dayjs from "dayjs"

// 格式化日期
const formatDate = (date: string | number) => {
  return dayjs(date).format("YYYY-MM-DD HH:mm:ss")
}
```

#### 2.5.2 缓存操作

```typescript
import { getToken, setToken, removeToken } from "@@/utils/cache/cookies"
import { getItem, setItem, removeItem } from "@@/utils/cache/local-storage"

// Cookie 操作（用于 Token）
const token = getToken()
setToken("your-token")
removeToken()

// LocalStorage 操作
const value = getItem("key")
setItem("key", { data: "value" })
removeItem("key")
```

---

## 3. 后端开发指南

### 3.1 技术栈

- **语言**: Go 1.24.3
- **框架**: Gin 1.10.1
- **ORM**: Xorm 1.3.10
- **数据库**: MySQL + Redis
- **JWT**: golang-jwt/jwt/v5
- **日志**: go-zero/core/logx

### 3.2 目录结构

```
server/internal/server/
├── model/          # 数据模型（请求/响应结构体，平铺结构）
│   ├── auth.go           # 认证相关
│   ├── twofa.go          # 2FA相关
│   ├── merchant.go       # 商户相关
│   ├── cloud_aliyun.go   # 阿里云相关
│   ├── common.go         # 通用类型（分页等）
│   └── admin.go          # 管理员相关
├── router/         # 路由定义（按模块分目录）
│   ├── auth/             # 认证路由模块
│   │   ├── auth.go       # 登录认证
│   │   └── twofa.go      # 2FA管理
│   ├── merchant/         # 商户路由模块
│   │   └── merchant.go
│   └── cloud_aliyun/     # 阿里云路由模块
│       └── cloud_aliyun.go
├── service/        # 业务逻辑（按模块分目录）
│   ├── auth/             # 认证服务模块
│   │   ├── auth.go
│   │   ├── auth_test.go
│   │   └── twofa.go
│   ├── merchant/         # 商户服务模块
│   │   └── merchant.go
│   └── cloud_aliyun/     # 阿里云服务模块
│       └── cloud_aliyun.go
├── middleware/     # 中间件
│   └── middleware.go
├── cloud/          # 云服务SDK封装
│   └── aliyun/           # 阿里云SDK操作
├── cfg/            # 配置
├── static/         # 静态文件（前端构建产物）
└── serve.go        # 服务启动入口

server/pkg/
├── dbs/            # 数据库连接
├── result/         # 统一响应格式
├── token_manager/  # Token 管理
└── entity/         # 数据库实体
```

### 3.3 开发规范

#### 3.3.1 Model 定义

**位置**: `server/internal/server/model/`

**说明**: Model层采用平铺结构，所有模型文件直接放在model目录下，按业务模块分文件。

**请求/响应结构体**:

```go
package model

// ========== 基础类型 ==========

// 分页基础结构
type Pagination struct {
    Page int `json:"page" form:"page"` // 页码
    Size int `json:"size" form:"size"` // 每页数量
}

// ========== 用户模块 ==========

// 查询用户请求
type QueryUsersReq struct {
    Pagination
    Id         int64  `json:"id" form:"id"`
    Phone      string `json:"phone" form:"phone"`
    Username   string `json:"username" form:"username"`
    RegisterIp string `json:"register_ip" form:"register_ip"`
    Order      string `json:"order" form:"order"` // desc, asc
}

// 创建用户请求
type CreateUserReq struct {
    Username  string `json:"username" binding:"required"`
    Phone     string `json:"phone" binding:"required"`
    Password  string `json:"password" binding:"required"`
    FirstName string `json:"first_name"`
    LastName  string `json:"last_name"`
}

// 更新用户请求
type UpdateUserReq struct {
    FirstName         string `json:"first_name"`
    LastName          string `json:"last_name"`
    Username          string `json:"username"`
    Verified          *int   `json:"verified"`
    Premium           *int   `json:"premium"`
    PremiumExpireDate *int64 `json:"premium_expire_date"`
}

// 用户响应
type UserResp struct {
    Id                int64  `json:"id"`
    Username          string `json:"username"`
    Phone             string `json:"phone"`
    FirstName         string `json:"first_name"`
    LastName          string `json:"last_name"`
    Verified          int    `json:"verified"`
    Premium           int    `json:"premium"`
    PremiumExpireDate int64  `json:"premium_expire_date"`
    RegisterIp        string `json:"register_ip"`
    CreatedAt         string `json:"created_at"`
}

// 用户列表响应（带分页）
type QueryUsersResponse struct {
    List  []UserResp `json:"list"`
    Total int        `json:"total"`
}
```

**命名规范**:

- 请求结构体: `{操作}{模块}Req` (如 `QueryUsersReq`, `CreateUserReq`)
- 响应结构体: `{模块}Resp` (如 `UserResp`)
- 列表响应: `Query{模块}Response` (如 `QueryUsersResponse`)

**字段标签**:

```go
// form: URL 参数绑定（GET 请求）
Field string `json:"field" form:"field"`

// binding: 参数验证
Field string `json:"field" binding:"required"`

// uri: 路径参数绑定
Id int64 `uri:"id" binding:"required"`
```

#### 3.3.2 Router 定义

**位置**: `server/internal/server/router/{模块名}/`

**说明**: Router层按业务模块分目录，每个模块提供统一的 `Routes()` 注册函数。

**目录结构**:
```
router/
├── auth/          # 认证模块路由
│   ├── auth.go
│   └── twofa.go
├── merchant/      # 商户模块路由
│   └── merchant.go
└── cloud_aliyun/  # 阿里云模块路由
    └── cloud_aliyun.go
```

**示例**: `router/merchant/merchant.go`

```go
package merchant

import (
    "server/internal/server/middleware"
    "server/internal/server/model"
    merchantService "server/internal/server/service/merchant"  // 使用别名避免包名冲突
    "server/pkg/result"
    "strconv"
    
    "github.com/gin-gonic/gin"
)

// Routes 注册商户相关路由
func Routes(gi gin.IRouter) {
    group := gi.Group("merchant")
    group.Use(middleware.Authorization) // 需要认证
    
    group.GET("", listMerchant)         // GET /merchant - 查询商户列表
    group.POST("", createMerchant)      // POST /merchant - 创建商户
    group.PUT(":id", updateMerchant)    // PUT /merchant/:id - 更新商户
    group.DELETE(":id", deleteMerchant) // DELETE /merchant/:id - 删除商户
}

// 查询商户列表
func listMerchant(ctx *gin.Context) {
    page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
    size, _ := strconv.Atoi(ctx.DefaultQuery("size", "10"))
    name := ctx.Query("name")
    orderBy := ctx.DefaultQuery("order", "id desc")
    
    merchantList, total, err := merchantService.ListMerchant(page, size, name, orderBy, 0)
    if err != nil {
        result.GErr(ctx, err)
        return
    }
    
    result.GOK(ctx, gin.H{
        "list":  merchantList,
        "total": total,
    })
}

// 获取用户详情
func getUserDetail(ctx *gin.Context) {
    idStr := ctx.Param("id")
    id, err := strconv.ParseInt(idStr, 10, 64)
    if err != nil {
        result.GParamErr(ctx, err)
        return
    }
    
    data, err := service.GetUserDetail(id)
    if err != nil {
        result.GErr(ctx, err)
        return
    }
    
    result.GOK(ctx, data)
}

// 创建用户
func createUser(ctx *gin.Context) {
    var req model.CreateUserReq
    if err := ctx.ShouldBindJSON(&req); err != nil {
        result.GParamErr(ctx, err)
        return
    }
    
    data, err := service.CreateUser(req)
    if err != nil {
        result.GErr(ctx, err)
        return
    }
    
    result.GOK(ctx, data)
}

// 更新用户
func updateUser(ctx *gin.Context) {
    idStr := ctx.Param("id")
    id, err := strconv.ParseInt(idStr, 10, 64)
    if err != nil {
        result.GParamErr(ctx, err)
        return
    }
    
    var req model.UpdateUserReq
    if err := ctx.ShouldBindJSON(&req); err != nil {
        result.GParamErr(ctx, err)
        return
    }
    
    err = service.UpdateUser(id, req)
    if err != nil {
        result.GErr(ctx, err)
        return
    }
    
    result.GOK(ctx, nil)
}

// 删除用户
func deleteUser(ctx *gin.Context) {
    idStr := ctx.Param("id")
    id, err := strconv.ParseInt(idStr, 10, 64)
    if err != nil {
        result.GParamErr(ctx, err)
        return
    }
    
    err = service.DeleteUser(id)
    if err != nil {
        result.GErr(ctx, err)
        return
    }
    
    result.GOK(ctx, nil)
}
```

**路由注册**: `server/internal/server/serve.go`

```go
package server

import (
    "context"
    "server/internal/server/cfg"
    "server/internal/server/middleware"
    "server/internal/server/router/auth"
    "server/internal/server/router/merchant"
    "server/internal/server/router/cloud_aliyun"
    "server/pkg/dbs"
    "server/pkg/token_manager"
    
    "github.com/gin-gonic/gin"
)

func Serve(ctx context.Context) {
    // 初始化数据库和Token管理器
    dbs.InitMysql(cfg.C.Mysql, &dbs.DBAdmin)
    dbs.InitRedis(cfg.C.Redis)
    token_manager.Init()
    
    ge := gin.Default()
    group := ge.Group("/server/v1")
    group.Use(middleware.LogRequest) // 请求日志
    
    // 注册各模块路由
    auth.Routes(group)         // 认证相关（登录、2FA）
    merchant.Routes(group)     // 商户管理
    cloud_aliyun.Routes(group) // 阿里云管理
    
    ge.Run(cfg.C.ListenOn)
}
```

#### 3.3.3 Service 实现

**位置**: `server/internal/server/service/{模块名}/`

**说明**: Service层按业务模块分目录，每个模块独立管理自己的业务逻辑。

**目录结构**:
```
service/
├── auth/          # 认证服务
│   ├── auth.go
│   ├── auth_test.go
│   └── twofa.go
├── merchant/      # 商户服务
│   └── merchant.go
└── cloud_aliyun/  # 阿里云服务
    └── cloud_aliyun.go
```

**示例**: `service/merchant/merchant.go`

```go
package merchant

import (
    "errors"
    "server/internal/server/model"
    "server/pkg/dbs"
    "server/pkg/entity"
    
    "github.com/zeromicro/go-zero/core/logx"
)

// QueryUsers 查询用户列表
func QueryUsers(req model.QueryUsersReq) (model.QueryUsersResponse, error) {
    var resp model.QueryUsersResponse
    
    // 构建查询
    session := dbs.DBAdmin.Table("users")
    
    // 条件过滤
    if req.Id > 0 {
        session = session.Where("id = ?", req.Id)
    }
    if req.Username != "" {
        session = session.Where("username LIKE ?", "%"+req.Username+"%")
    }
    if req.Phone != "" {
        session = session.Where("phone = ?", req.Phone)
    }
    
    // 排序
    if req.Order == "asc" {
        session = session.Asc("id")
    } else {
        session = session.Desc("id")
    }
    
    // 总数
    total, err := session.Count(&entity.User{})
    if err != nil {
        logx.Errorf("count users err: %+v", err)
        return resp, err
    }
    resp.Total = int(total)
    
    // 分页查询
    offset := (req.Page - 1) * req.Size
    var users []entity.User
    err = session.Limit(req.Size, offset).Find(&users)
    if err != nil {
        logx.Errorf("query users err: %+v", err)
        return resp, err
    }
    
    // 转换为响应格式
    for _, u := range users {
        resp.List = append(resp.List, model.UserResp{
            Id:                u.Id,
            Username:          u.Username,
            Phone:             u.Phone,
            FirstName:         u.FirstName,
            LastName:          u.LastName,
            Verified:          u.Verified,
            Premium:           u.Premium,
            PremiumExpireDate: u.PremiumExpireDate,
            RegisterIp:        u.RegisterIp,
            CreatedAt:         u.CreatedAt.Format("2006-01-02 15:04:05"),
        })
    }
    
    return resp, nil
}

// GetUserDetail 获取用户详情
func GetUserDetail(id int64) (model.UserResp, error) {
    var resp model.UserResp
    var user entity.User
    
    has, err := dbs.DBAdmin.Where("id = ?", id).Get(&user)
    if err != nil {
        logx.Errorf("get user err: %+v", err)
        return resp, err
    }
    if !has {
        return resp, errors.New("用户不存在")
    }
    
    resp = model.UserResp{
        Id:                user.Id,
        Username:          user.Username,
        Phone:             user.Phone,
        FirstName:         user.FirstName,
        LastName:          user.LastName,
        // ... 其他字段
    }
    
    return resp, nil
}

// CreateUser 创建用户
func CreateUser(req model.CreateUserReq) (int64, error) {
    user := entity.User{
        Username:  req.Username,
        Phone:     req.Phone,
        Password:  req.Password, // 实际应该加密
        FirstName: req.FirstName,
        LastName:  req.LastName,
    }
    
    affected, err := dbs.DBAdmin.Insert(&user)
    if err != nil {
        logx.Errorf("create user err: %+v", err)
        return 0, err
    }
    if affected == 0 {
        return 0, errors.New("创建失败")
    }
    
    return user.Id, nil
}

// UpdateUser 更新用户
func UpdateUser(id int64, req model.UpdateUserReq) error {
    updates := make(map[string]interface{})
    
    if req.Username != "" {
        updates["username"] = req.Username
    }
    if req.FirstName != "" {
        updates["first_name"] = req.FirstName
    }
    if req.Verified != nil {
        updates["verified"] = *req.Verified
    }
    
    if len(updates) == 0 {
        return errors.New("没有需要更新的字段")
    }
    
    affected, err := dbs.DBAdmin.Table("users").
        Where("id = ?", id).
        Update(updates)
    if err != nil {
        logx.Errorf("update user err: %+v", err)
        return err
    }
    if affected == 0 {
        return errors.New("用户不存在或无变更")
    }
    
    return nil
}

// DeleteUser 删除用户
func DeleteUser(id int64) error {
    affected, err := dbs.DBAdmin.Where("id = ?", id).Delete(&entity.User{})
    if err != nil {
        logx.Errorf("delete user err: %+v", err)
        return err
    }
    if affected == 0 {
        return errors.New("用户不存在")
    }
    
    return nil
}
```

#### 3.3.4 包导入规范

**重要**: 当router包名与service包名相同时，需要使用导入别名：

```go
package merchant  // router包名

import (
    "server/internal/server/model"
    merchantService "server/internal/server/service/merchant"  // 使用别名
    "github.com/gin-gonic/gin"
)

func listMerchant(c *gin.Context) {
    // 调用service时使用别名
    data, err := merchantService.ListMerchant(page, size, name, orderBy, expiringSoon)
    // ...
}
```

**常用别名**:
```go
authService "server/internal/server/service/auth"
merchantService "server/internal/server/service/merchant"
cloudService "server/internal/server/service/cloud_aliyun"
```

#### 3.3.5 统一响应格式

**位置**: `server/pkg/result/gin.go`

```go
package result

import (
    "net/http"
    "github.com/gin-gonic/gin"
)

// GOK 成功响应
func GOK(c *gin.Context, data any) {
    GResult(c, 200, data)
}

// GErr 错误响应
func GErr(c *gin.Context, err error) {
    GResult(c, 400, nil, err.Error())
}

// GParamErr 参数错误
func GParamErr(c *gin.Context, err error) {
    GResult(c, 601, nil, err.Error())
}

// GResult 统一响应格式
func GResult(c *gin.Context, code int, data any, msg ...string) {
    c.Abort()
    var tmpMsg string
    if len(msg) > 0 {
        tmpMsg = msg[0]
    }
    c.JSON(http.StatusOK, gin.H{
        "code":    code,
        "data":    data,
        "message": tmpMsg,
    })
}

// GAuthErr 认证失败
func GAuthErr(c *gin.Context) {
    c.Abort()
    c.JSON(http.StatusUnauthorized, nil)
}
```

**使用方式**:

```go
// 成功
result.GOK(ctx, data)

// 错误
result.GErr(ctx, errors.New("操作失败"))

// 参数错误
result.GParamErr(ctx, err)

// 自定义
result.GResult(ctx, 500, nil, "服务器错误")
```

#### 3.3.6 中间件

**认证中间件**: `server/internal/server/middleware/middleware.go`

```go
package middleware

import (
    "server/pkg/result"
    "server/pkg/token_manager"
    "strings"
    
    "github.com/gin-gonic/gin"
)

const (
    contextKeyUserId   = "user_id"
    contextKeyUsername = "username"
    contextKeyTwoFA    = "two_fa"
)

// Authorization 认证中间件
func Authorization(c *gin.Context) {
    // 获取 Token
    authHeader := c.GetHeader("Authorization")
    if authHeader == "" {
        result.GAuthErr(c)
        return
    }
    
    // 验证 Bearer Token
    parts := strings.Split(authHeader, " ")
    if len(parts) != 2 || parts[0] != "Bearer" {
        result.GAuthErr(c)
        return
    }
    
    // 解析 Token
    token := parts[1]
    claims, err := token_manager.ParseToken(token)
    if err != nil {
        result.GAuthErr(c)
        return
    }
    
    // 存储用户信息到上下文
    c.Set(contextKeyUserId, claims.UserId)
    c.Set(contextKeyUsername, claims.Username)
    c.Set(contextKeyTwoFA, claims.TwoFA)
    
    c.Next()
}

// GetUserId 获取当前用户 ID
func GetUserId(c *gin.Context) int {
    val, _ := c.Get(contextKeyUserId)
    userId, _ := val.(int)
    return userId
}

// GetUsername 获取当前用户名
func GetUsername(c *gin.Context) string {
    val, _ := c.Get(contextKeyUsername)
    username, _ := val.(string)
    return username
}
```

**日志中间件**:

```go
// LogRequest 请求日志中间件
func LogRequest(c *gin.Context) {
    start := time.Now()
    path := c.Request.URL.Path
    
    c.Next()
    
    latency := time.Since(start)
    status := c.Writer.Status()
    
    logx.Infof("[%s] %s %d %v",
        c.Request.Method,
        path,
        status,
        latency,
    )
}
```

### 3.4 数据库操作

#### 3.4.1 Xorm 常用操作

```go
// 查询单条
var user entity.User
has, err := dbs.DBAdmin.Where("id = ?", id).Get(&user)

// 查询多条
var users []entity.User
err := dbs.DBAdmin.Where("status = ?", 1).Find(&users)

// 分页查询
err := dbs.DBAdmin.Limit(pageSize, offset).Find(&users)

// 统计
total, err := dbs.DBAdmin.Where("status = ?", 1).Count(&entity.User{})

// 插入
affected, err := dbs.DBAdmin.Insert(&user)

// 更新（根据主键）
affected, err := dbs.DBAdmin.ID(id).Update(&user)

// 更新（指定字段）
affected, err := dbs.DBAdmin.Table("users").
    Where("id = ?", id).
    Update(map[string]interface{}{
        "username": "new_name",
        "verified": 1,
    })

// 删除
affected, err := dbs.DBAdmin.Where("id = ?", id).Delete(&entity.User{})

// 事务
session := dbs.DBAdmin.NewSession()
defer session.Close()
if err := session.Begin(); err != nil {
    return err
}
// ... 执行多个操作
if err := session.Commit(); err != nil {
    session.Rollback()
    return err
}
```

#### 3.4.2 Xorm 重要注意事项 ⚠️

**Session 只能使用一次！**

❌ **错误示例**（会导致查询条件失效）：
```go
session := dbs.DBAdmin.Table("users")
session = session.Where("status = ?", 1)

// Count 消费了 session
total, err := session.Count(&entity.User{})

// Find 时条件失效，查询所有数据！
err = session.Find(&users)  // ❌ BUG: 条件失效
```

✅ **正确方式一：使用 FindAndCount**
```go
session := dbs.DBAdmin.Table("users")
session = session.Where("status = ?", 1)

// 一次性完成计数和查询
total, err := session.Limit(10, 0).FindAndCount(&users)  // ✅ 正确
```

✅ **正确方式二：克隆 Session**
```go
session := dbs.DBAdmin.Table("users")
session = session.Where("status = ?", 1)

// 先计数
countSession := session.Clone()
total, err := countSession.Count(&entity.User{})

// 再查询
err = session.Find(&users)  // ✅ 正确
```

✅ **正确方式三：分开构建**
```go
// 计数
total, err := dbs.DBAdmin.Table("users").Where("status = ?", 1).Count(&entity.User{})

// 查询
var users []entity.User
err = dbs.DBAdmin.Table("users").Where("status = ?", 1).Find(&users)
```

**推荐：** 使用 `FindAndCount()`，一次完成，性能更好！

#### 3.4.3 Redis 操作

```go
import "server/pkg/dbs"

ctx := context.Background()

// 设置值
err := dbs.Rds().Set(ctx, "key", "value", time.Hour).Err()

// 获取值
val, err := dbs.Rds().Get(ctx, "key").Result()

// 删除
err := dbs.Rds().Del(ctx, "key").Err()

// 自增
count, err := dbs.Rds().Incr(ctx, "counter").Result()

// 设置过期时间
err := dbs.Rds().Expire(ctx, "key", time.Hour).Err()

// 检查是否存在
exists, err := dbs.Rds().Exists(ctx, "key").Result()
```

---

## 4. 完整开发流程示例

### 场景: 新增"商品管理"功能模块

包括：商品列表、创建商品、编辑商品、删除商品

---

### 4.1 后端开发

#### Step 1: 定义数据模型

**文件**: `server/internal/server/model/products.go`

```go
package model

// ========== 商品管理 ==========

// 查询商品请求
type QueryProductsReq struct {
    Pagination
    Name     string `json:"name" form:"name"`         // 商品名称（模糊查询）
    Category string `json:"category" form:"category"` // 分类
    Status   *int   `json:"status" form:"status"`     // 状态（0:下架 1:上架）
}

// 创建商品请求
type CreateProductReq struct {
    Name        string  `json:"name" binding:"required"`
    Description string  `json:"description"`
    Price       float64 `json:"price" binding:"required,gt=0"`
    Stock       int     `json:"stock" binding:"required,gte=0"`
    Category    string  `json:"category" binding:"required"`
    Image       string  `json:"image"`
}

// 更新商品请求
type UpdateProductReq struct {
    Name        string   `json:"name"`
    Description string   `json:"description"`
    Price       *float64 `json:"price"`
    Stock       *int     `json:"stock"`
    Category    string   `json:"category"`
    Image       string   `json:"image"`
    Status      *int     `json:"status"`
}

// 商品响应
type ProductResp struct {
    Id          int64   `json:"id"`
    Name        string  `json:"name"`
    Description string  `json:"description"`
    Price       float64 `json:"price"`
    Stock       int     `json:"stock"`
    Category    string  `json:"category"`
    Image       string  `json:"image"`
    Status      int     `json:"status"`
    CreatedAt   string  `json:"created_at"`
    UpdatedAt   string  `json:"updated_at"`
}

// 商品列表响应
type QueryProductsResponse struct {
    List  []ProductResp `json:"list"`
    Total int           `json:"total"`
}
```

#### Step 2: 实现 Service

**文件**: `server/internal/server/service/products/products.go`（创建新目录）

**说明**: Service层按模块分目录，与router结构保持一致。

```go
package products

import (
    "errors"
    "server/internal/server/model"
    "server/pkg/dbs"
    "server/pkg/entity"
    
    "github.com/zeromicro/go-zero/core/logx"
)

// QueryProducts 查询商品列表
func QueryProducts(req model.QueryProductsReq) (model.QueryProductsResponse, error) {
    var resp model.QueryProductsResponse
    
    session := dbs.DBAdmin.Table("products")
    
    // 条件过滤
    if req.Name != "" {
        session = session.Where("name LIKE ?", "%"+req.Name+"%")
    }
    if req.Category != "" {
        session = session.Where("category = ?", req.Category)
    }
    if req.Status != nil {
        session = session.Where("status = ?", *req.Status)
    }
    
    // 总数
    total, err := session.Count(&entity.Product{})
    if err != nil {
        logx.Errorf("count products err: %+v", err)
        return resp, err
    }
    resp.Total = int(total)
    
    // 分页查询
    offset := (req.Page - 1) * req.Size
    var products []entity.Product
    err = session.Desc("id").Limit(req.Size, offset).Find(&products)
    if err != nil {
        logx.Errorf("query products err: %+v", err)
        return resp, err
    }
    
    // 转换响应
    for _, p := range products {
        resp.List = append(resp.List, model.ProductResp{
            Id:          p.Id,
            Name:        p.Name,
            Description: p.Description,
            Price:       p.Price,
            Stock:       p.Stock,
            Category:    p.Category,
            Image:       p.Image,
            Status:      p.Status,
            CreatedAt:   p.CreatedAt.Format("2006-01-02 15:04:05"),
            UpdatedAt:   p.UpdatedAt.Format("2006-01-02 15:04:05"),
        })
    }
    
    return resp, nil
}

// CreateProduct 创建商品
func CreateProduct(req model.CreateProductReq) (int64, error) {
    product := entity.Product{
        Name:        req.Name,
        Description: req.Description,
        Price:       req.Price,
        Stock:       req.Stock,
        Category:    req.Category,
        Image:       req.Image,
        Status:      1, // 默认上架
    }
    
    affected, err := dbs.DBAdmin.Insert(&product)
    if err != nil {
        logx.Errorf("create product err: %+v", err)
        return 0, err
    }
    if affected == 0 {
        return 0, errors.New("创建失败")
    }
    
    return product.Id, nil
}

// UpdateProduct 更新商品
func UpdateProduct(id int64, req model.UpdateProductReq) error {
    updates := make(map[string]interface{})
    
    if req.Name != "" {
        updates["name"] = req.Name
    }
    if req.Description != "" {
        updates["description"] = req.Description
    }
    if req.Price != nil {
        updates["price"] = *req.Price
    }
    if req.Stock != nil {
        updates["stock"] = *req.Stock
    }
    if req.Category != "" {
        updates["category"] = req.Category
    }
    if req.Image != "" {
        updates["image"] = req.Image
    }
    if req.Status != nil {
        updates["status"] = *req.Status
    }
    
    if len(updates) == 0 {
        return errors.New("没有需要更新的字段")
    }
    
    affected, err := dbs.DBAdmin.Table("products").
        Where("id = ?", id).
        Update(updates)
    if err != nil {
        logx.Errorf("update product err: %+v", err)
        return err
    }
    if affected == 0 {
        return errors.New("商品不存在")
    }
    
    return nil
}

// DeleteProduct 删除商品
func DeleteProduct(id int64) error {
    affected, err := dbs.DBAdmin.Where("id = ?", id).Delete(&entity.Product{})
    if err != nil {
        logx.Errorf("delete product err: %+v", err)
        return err
    }
    if affected == 0 {
        return errors.New("商品不存在")
    }
    
    return nil
}
```

#### Step 3: 定义路由

**文件**: `server/internal/server/router/products/products.go`（创建新目录）

```go
package products

import (
    "server/internal/server/middleware"
    "server/internal/server/model"
    productService "server/internal/server/service/products"  // 使用别名
    "server/pkg/result"
    "strconv"
    
    "github.com/gin-gonic/gin"
)

// Routes 商品管理路由注册
func Routes(gi gin.IRouter) {
    group := gi.Group("products")
    group.Use(middleware.Authorization)
    
    group.GET("", queryProducts)
    group.POST("", createProduct)
    group.PUT(":id", updateProduct)
    group.DELETE(":id", deleteProduct)
}

func queryProducts(ctx *gin.Context) {
    var req model.QueryProductsReq
    if err := ctx.ShouldBindQuery(&req); err != nil {
        result.GParamErr(ctx, err)
        return
    }
    
    data, err := productService.QueryProducts(req)
    if err != nil {
        result.GErr(ctx, err)
        return
    }
    
    result.GOK(ctx, data)
}

func createProduct(ctx *gin.Context) {
    var req model.CreateProductReq
    if err := ctx.ShouldBindJSON(&req); err != nil {
        result.GParamErr(ctx, err)
        return
    }
    
    id, err := productService.CreateProduct(req)
    if err != nil {
        result.GErr(ctx, err)
        return
    }
    
    result.GOK(ctx, map[string]interface{}{"id": id})
}

func updateProduct(ctx *gin.Context) {
    id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
    if err != nil {
        result.GParamErr(ctx, err)
        return
    }
    
    var req model.UpdateProductReq
    if err := ctx.ShouldBindJSON(&req); err != nil {
        result.GParamErr(ctx, err)
        return
    }
    
    err = productService.UpdateProduct(id, req)
    if err != nil {
        result.GErr(ctx, err)
        return
    }
    
    result.GOK(ctx, nil)
}

func deleteProduct(ctx *gin.Context) {
    id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
    if err != nil {
        result.GParamErr(ctx, err)
        return
    }
    
    err = productService.DeleteProduct(id)
    if err != nil {
        result.GErr(ctx, err)
        return
    }
    
    result.GOK(ctx, nil)
}
```

#### Step 4: 注册路由

**文件**: `server/internal/server/serve.go`

```go
package server

import (
    "context"
    "server/internal/server/router/auth"
    "server/internal/server/router/merchant"
    "server/internal/server/router/products"  // ← 导入新模块
    "github.com/gin-gonic/gin"
)

func Serve(ctx context.Context) {
    // ... 初始化代码 ...
    
    ge := gin.Default()
    group := ge.Group("/server/v1")
    
    auth.Routes(group)      // 认证路由
    merchant.Routes(group)  // 商户路由
    products.Routes(group)  // ← 新增路由注册
    
    ge.Run(cfg.C.ListenOn)
}
```

**注意**: 每个模块使用独立的 `Routes()` 函数进行注册。

---

### 4.2 前端开发

#### Step 1: 定义 API 类型

**文件**: `front/src/common/apis/products/type.ts`

```typescript
import type { ApiResponseData } from "@/common/apis/type"

// 分页
export interface Pagination {
  page: number
  size: number
}

// 查询请求
export interface QueryProductsReq extends Pagination {
  name?: string
  category?: string
  status?: number
}

// 创建请求
export interface CreateProductReq {
  name: string
  description?: string
  price: number
  stock: number
  category: string
  image?: string
}

// 更新请求
export interface UpdateProductReq {
  name?: string
  description?: string
  price?: number
  stock?: number
  category?: string
  image?: string
  status?: number
}

// 商品响应
export interface ProductResp {
  id: number
  name: string
  description: string
  price: number
  stock: number
  category: string
  image: string
  status: number
  created_at: string
  updated_at: string
}

// 列表响应
export interface QueryProductsResponse {
  list: ProductResp[]
  total: number
}

// API 响应类型
export type QueryProductsResponseData = ApiResponseData<QueryProductsResponse>
export type ProductDetailResponseData = ApiResponseData<ProductResp>
```

#### Step 2: 定义 API 函数

**文件**: `front/src/common/apis/products/index.ts`

```typescript
import type * as Products from "./type"
import { request } from "@/http/axios"

/** 获取商品列表 */
export function getProductList(params: Products.QueryProductsReq) {
  return request<Products.QueryProductsResponseData>({
    url: "products",
    method: "get",
    params
  })
}

/** 创建商品 */
export function createProduct(data: Products.CreateProductReq) {
  return request({
    url: "products",
    method: "post",
    data
  })
}

/** 更新商品 */
export function updateProduct(id: number, data: Products.UpdateProductReq) {
  return request({
    url: `products/${id}`,
    method: "put",
    data
  })
}

/** 删除商品 */
export function deleteProduct(id: number) {
  return request({
    url: `products/${id}`,
    method: "delete"
  })
}
```

#### Step 3: 创建页面组件

**文件**: `front/src/pages/products/list.vue`

```vue
<script lang="ts" setup>
import type { VxeGridInstance, VxeGridProps, VxeFormInstance, VxeFormProps, VxeModalInstance, VxeModalProps } from "vxe-table"
import type { ProductResp } from "@@/apis/products/type"
import { getProductList, createProduct, updateProduct, deleteProduct } from "@@/apis/products"

defineOptions({
  name: "ProductList"
})

// ========== VXE Grid 配置 ==========
const xGridDom = ref<VxeGridInstance>()
const xGridOpt: VxeGridProps = reactive({
  loading: true,
  autoResize: true,
  pagerConfig: {
    align: "right"
  },
  formConfig: {
    items: [
      {
        field: "name",
        itemRender: {
          name: "$input",
          props: { placeholder: "商品名称", clearable: true }
        }
      },
      {
        field: "category",
        itemRender: {
          name: "$input",
          props: { placeholder: "分类", clearable: true }
        }
      },
      {
        itemRender: {
          name: "$buttons",
          children: [
            { props: { type: "submit", content: "查询", status: "primary" } },
            { props: { type: "reset", content: "重置" } }
          ]
        }
      }
    ]
  },
  toolbarConfig: {
    refresh: true,
    custom: true,
    slots: { buttons: "toolbar-btns" }
  },
  columns: [
    { type: "checkbox", width: "50px" },
    { field: "name", title: "商品名称", width: 200 },
    { field: "description", title: "描述", showOverflow: true },
    { field: "price", title: "价格", width: 100 },
    { field: "stock", title: "库存", width: 100 },
    { field: "category", title: "分类", width: 120 },
    { 
      field: "status", 
      title: "状态", 
      width: 100,
      slots: { default: "status-slot" }
    },
    { field: "created_at", title: "创建时间", width: 180 },
    {
      title: "操作",
      width: "150px",
      fixed: "right",
      slots: { default: "row-operate" }
    }
  ],
  proxyConfig: {
    seq: true,
    form: true,
    autoLoad: true,
    props: { total: "total" },
    ajax: {
      query: ({ page, form }) => {
        xGridOpt.loading = true
        return new Promise((resolve) => {
          const params = {
            name: form.name || "",
            category: form.category || "",
            size: page.pageSize,
            page: page.currentPage
          }
          getProductList(params).then((res) => {
            xGridOpt.loading = false
            resolve({
              total: res.data.total,
              result: res.data.list
            })
          }).catch(() => {
            xGridOpt.loading = false
          })
        })
      }
    }
  }
})

// ========== Modal & Form 配置 ==========
const xModalDom = ref<VxeModalInstance>()
const xFormDom = ref<VxeFormInstance>()

const xModalOpt: VxeModalProps = reactive({
  title: "",
  showClose: true,
  escClosable: true,
  maskClosable: true,
  beforeHideMethod: () => {
    xFormDom.value?.clearValidate()
    return Promise.resolve()
  }
})

const xFormOpt: VxeFormProps = reactive({
  span: 24,
  titleWidth: "100px",
  loading: false,
  titleColon: false,
  data: {
    name: "",
    description: "",
    price: 0,
    stock: 0,
    category: "",
    image: ""
  },
  items: [
    {
      field: "name",
      title: "商品名称",
      itemRender: {
        name: "$input",
        props: { placeholder: "请输入" }
      }
    },
    {
      field: "description",
      title: "商品描述",
      itemRender: {
        name: "$textarea",
        props: { placeholder: "请输入", rows: 3 }
      }
    },
    {
      field: "price",
      title: "价格",
      itemRender: {
        name: "$input",
        props: { type: "number", placeholder: "请输入" }
      }
    },
    {
      field: "stock",
      title: "库存",
      itemRender: {
        name: "$input",
        props: { type: "number", placeholder: "请输入" }
      }
    },
    {
      field: "category",
      title: "分类",
      itemRender: {
        name: "$input",
        props: { placeholder: "请输入" }
      }
    },
    {
      align: "right",
      itemRender: {
        name: "$buttons",
        children: [
          {
            props: { content: "取消" },
            events: { click: () => xModalDom.value?.close() }
          },
          {
            props: { type: "submit", content: "确定", status: "primary" },
            events: { click: () => crudStore.onSubmitForm() }
          }
        ]
      }
    }
  ],
  rules: {
    name: [{ required: true, message: "请输入商品名称" }],
    price: [{ required: true, message: "请输入价格" }],
    stock: [{ required: true, message: "请输入库存" }],
    category: [{ required: true, message: "请输入分类" }]
  }
})

// ========== CRUD 操作 ==========
const crudStore = reactive({
  isUpdate: false,
  currentId: 0,
  
  commitQuery: () => xGridDom.value?.commitProxy("query"),
  
  onShowModal: (row?: ProductResp) => {
    if (row) {
      crudStore.isUpdate = true
      crudStore.currentId = row.id
      xModalOpt.title = "编辑商品"
      xFormOpt.data = {
        name: row.name,
        description: row.description,
        price: row.price,
        stock: row.stock,
        category: row.category,
        image: row.image
      }
    } else {
      crudStore.isUpdate = false
      crudStore.currentId = 0
      xModalOpt.title = "新增商品"
    }
    xModalDom.value?.open()
    nextTick(() => {
      !crudStore.isUpdate && xFormDom.value?.reset()
      xFormDom.value?.clearValidate()
    })
  },
  
  onSubmitForm: () => {
    if (xFormOpt.loading) return
    xFormDom.value?.validate((errMap) => {
      if (errMap) return
      xFormOpt.loading = true
      
      const apiCall = crudStore.isUpdate
        ? updateProduct(crudStore.currentId, xFormOpt.data)
        : createProduct(xFormOpt.data)
      
      apiCall.then(() => {
        xFormOpt.loading = false
        xModalDom.value?.close()
        ElMessage.success("操作成功")
        crudStore.commitQuery()
      }).catch(() => {
        xFormOpt.loading = false
      })
    })
  },
  
  onDelete: (row: ProductResp) => {
    ElMessageBox.confirm(
      `确定删除商品 "${row.name}" 吗？`,
      "提示",
      { type: "warning" }
    ).then(() => {
      deleteProduct(row.id).then(() => {
        ElMessage.success("删除成功")
        crudStore.commitQuery()
      })
    })
  }
})
</script>

<template>
  <div class="app-container">
    <!-- 表格 -->
    <vxe-grid ref="xGridDom" v-bind="xGridOpt">
      <!-- 工具栏按钮 -->
      <template #toolbar-btns>
        <vxe-button status="primary" icon="vxe-icon-add" @click="crudStore.onShowModal()">
          新增商品
        </vxe-button>
      </template>
      
      <!-- 状态列 -->
      <template #status-slot="{ row }">
        <el-tag :type="row.status === 1 ? 'success' : 'info'">
          {{ row.status === 1 ? '上架' : '下架' }}
        </el-tag>
      </template>
      
      <!-- 操作列 -->
      <template #row-operate="{ row }">
        <el-button link type="primary" @click="crudStore.onShowModal(row)">
          编辑
        </el-button>
        <el-button link type="danger" @click="crudStore.onDelete(row)">
          删除
        </el-button>
      </template>
    </vxe-grid>
    
    <!-- 弹窗 -->
    <vxe-modal ref="xModalDom" v-bind="xModalOpt">
      <vxe-form ref="xFormDom" v-bind="xFormOpt" />
    </vxe-modal>
  </div>
</template>

<style lang="scss" scoped>
.app-container {
  padding: 20px;
}
</style>
```

#### Step 4: 添加路由

**文件**: `front/src/router/index.ts`

```typescript
export const dynamicRoutes: RouteRecordRaw[] = [
  // ... 其他路由
  {
    path: "/products",
    component: Layouts,
    redirect: "/products/list",
    name: "Products",
    meta: {
      title: "商品管理",
      elIcon: "ShoppingCart",
      roles: ["admin"],
      alwaysShow: true
    },
    children: [
      {
        path: "list",
        component: () => import("@/pages/products/list.vue"),
        name: "ProductsList",
        meta: {
          title: "商品列表"
        }
      }
    ]
  }
]
```

---

## 5. 常见问题与最佳实践

### 5.1 前端

#### Q1: 何时使用 VXE Table，何时使用 Element Plus Table?

**A**: 
- **简单表格**: 使用 Element Plus Table
- **复杂表格**（大数据量、虚拟滚动、复杂表单、树形数据）: 使用 VXE Table

#### Q2: 如何处理大量数据的下拉选择？

**A**: 使用 `el-select-v2`（虚拟滚动）代替 `el-select`

```vue
<el-select-v2 
  v-model="value" 
  :options="options" 
  filterable 
  placeholder="请选择" 
/>
```

#### Q3: 如何全局处理 loading 状态？

**A**: 使用 `useFullscreenLoading` composable

```typescript
import { useFullscreenLoading } from "@@/composables/useFullscreenLoading"

const { openFullscreenLoading, closeFullscreenLoading } = useFullscreenLoading()

async function heavyTask() {
  openFullscreenLoading()
  try {
    await someApi()
  } finally {
    closeFullscreenLoading()
  }
}
```

#### Q4: 如何优雅地处理表单验证？

**A**: VXE Form 内置验证，Element Plus Form 使用 rules

```typescript
// VXE Form
rules: {
  username: [
    {
      required: true,
      validator: ({ itemValue }) => {
        if (!itemValue) return new Error("请输入")
        if (!itemValue.trim()) return new Error("空格无效")
      }
    }
  ]
}

// Element Plus Form
const rules = {
  username: [
    { required: true, message: "请输入用户名", trigger: "blur" },
    { min: 3, max: 20, message: "长度在 3 到 20 个字符", trigger: "blur" }
  ]
}
```

### 5.2 后端

#### Q1: 如何处理事务？

**A**:

```go
func TransferMoney(fromId, toId int64, amount float64) error {
    session := dbs.DBAdmin.NewSession()
    defer session.Close()
    
    if err := session.Begin(); err != nil {
        return err
    }
    
    // 操作1: 扣款
    _, err := session.Exec("UPDATE accounts SET balance = balance - ? WHERE id = ?", amount, fromId)
    if err != nil {
        session.Rollback()
        return err
    }
    
    // 操作2: 加款
    _, err = session.Exec("UPDATE accounts SET balance = balance + ? WHERE id = ?", amount, toId)
    if err != nil {
        session.Rollback()
        return err
    }
    
    return session.Commit()
}
```

#### Q2: 如何实现软删除？

**A**:

```go
// 软删除（更新 deleted 字段）
_, err := dbs.DBAdmin.Table("users").
    Where("id = ?", id).
    Update(map[string]interface{}{
        "deleted": 1,
        "deleted_at": time.Now(),
    })

// 查询时排除已删除
session.Where("deleted = ?", 0)
```

#### Q3: 如何记录操作日志？

**A**:

```go
import "github.com/zeromicro/go-zero/core/logx"

// 记录信息
logx.Infof("user %d login from %s", userId, ip)

// 记录错误
logx.Errorf("create user err: %+v", err)

// 记录慢查询
logx.Slowf("query took %v", duration)
```

#### Q4: 如何优化分页查询性能？

**A**:

```go
// 1. 添加索引
// 2. 使用子查询优化深分页
func QueryUsersOptimized(req model.QueryUsersReq) (model.QueryUsersResponse, error) {
    offset := (req.Page - 1) * req.Size
    
    // 先查 ID（只查主键，速度快）
    var ids []int64
    err := dbs.DBAdmin.Table("users").
        Where("status = ?", 1).
        Cols("id").
        Limit(req.Size, offset).
        Find(&ids)
    
    if len(ids) == 0 {
        return model.QueryUsersResponse{}, nil
    }
    
    // 再根据 ID 查完整数据
    var users []entity.User
    err = dbs.DBAdmin.In("id", ids).Find(&users)
    
    // ...
}
```

### 5.3 通用最佳实践

#### 1. **错误处理**

前端:
```typescript
try {
  const res = await someApi()
  // 处理成功
} catch (error) {
  // ElMessage 会自动显示错误信息（axios 拦截器处理）
  console.error(error)
}
```

后端:
```go
if err != nil {
    logx.Errorf("operation failed: %+v", err)
    return errors.New("操作失败")
}
```

#### 2. **命名规范**

- 前端组件: PascalCase (如 `UserList`)
- 前端文件: kebab-case (如 `user-list.vue`)
- 后端文件: snake_case (如 `user_service.go`)
- API 接口: RESTful 风格
  - `GET /users` - 列表
  - `GET /users/:id` - 详情
  - `POST /users` - 创建
  - `PUT /users/:id` - 更新
  - `DELETE /users/:id` - 删除

#### 3. **注释规范**

前端:
```typescript
/** 获取用户列表 */
export function getUserList(params: Users.QueryUsersReq) {
  // ...
}
```

后端:
```go
// QueryUsers 查询用户列表
func QueryUsers(req model.QueryUsersReq) (model.QueryUsersResponse, error) {
    // ...
}
```

#### 4. **代码组织**

- 按功能模块划分，不按技术类型划分
- 相关文件放在同一目录下
- 避免深层嵌套（不超过 3 层）

---

## 6. 开发工具链

### 6.1 前端

```bash
# 安装依赖
pnpm install

# 开发模式
pnpm dev

# 构建生产版本
pnpm build

# 代码检查
pnpm lint

# 单元测试
pnpm test
```

### 6.2 后端

```bash
# 运行服务
cd server/apps/server
go run main.go

# 构建
go build -o server main.go

# 热重载（需安装 air）
air
```

---

## 7. 总结

### 核心要点

1. **前端开发必看 demo 目录**: `front/src/pages/demo/`
2. **API 统一管理**: `front/src/common/apis/`
3. **复用 Composables**: `front/src/common/composables/`
4. **后端三层架构**: Model → Service → Router
5. **模块化组织**: Router和Service按业务模块分目录
6. **导入别名规范**: 使用别名避免包名冲突（如 `merchantService`）
7. **统一响应格式**: `result.GOK()` / `result.GErr()`
8. **类型安全**: 前后端都使用 TypeScript/Go 的类型系统

### 开发流程

```
需求分析 → 数据库设计 → 后端开发 → 前端开发 → 联调测试 → 上线部署
```

### 推荐工具

- **前端**: VSCode + Volar + TypeScript
- **后端**: GoLand / VSCode + Go 插件
- **API 测试**: Apifox / Postman
- **数据库**: MySQL Workbench / Navicat
- **版本控制**: Git

---

---

## 8. 项目架构说明

### 8.1 后端模块化架构

本项目采用**模块化分层架构**，提高代码组织性和可维护性：

#### 📁 目录组织原则

1. **Model层（平铺）**: 所有模型文件直接在 `model/` 下，按模块分文件
   - 优点：导入简单，适合中小型项目
   - 示例：`model/auth.go`, `model/merchant.go`

2. **Router层（模块化）**: 按业务模块分目录
   - 每个模块一个目录，提供 `Routes()` 注册函数
   - 示例：`router/auth/`, `router/merchant/`

3. **Service层（模块化）**: 与Router保持一致的目录结构
   - 每个模块一个目录，包含该模块的所有业务逻辑
   - 示例：`service/auth/`, `service/merchant/`

#### 🔧 开发新模块步骤

假设要开发"订单管理"模块：

1. **创建Model**: `model/order.go`
   ```go
   package model
   // 订单相关的结构体定义
   ```

2. **创建Service**: `service/order/order.go`
   ```go
   package order
   // 订单业务逻辑
   ```

3. **创建Router**: `router/order/order.go`
   ```go
   package order
   
   import (
       orderService "server/internal/server/service/order"  // 使用别名
   )
   
   func Routes(gi gin.IRouter) {
       // 路由注册
   }
   ```

4. **注册路由**: 在 `serve.go` 中添加
   ```go
   import "server/internal/server/router/order"
   
   order.Routes(group)
   ```

#### ⚠️ 重要注意事项

1. **包名冲突**: Router和Service包名相同时**必须使用别名**
   ```go
   // ✅ 正确
   merchantService "server/internal/server/service/merchant"
   
   // ❌ 错误 - 会导致编译错误
   "server/internal/server/service/merchant"
   ```

2. **统一命名**: 所有模块使用 `Routes()` 作为路由注册函数名

3. **Model不分目录**: 保持平铺结构，除非单个文件超过200行

---

**祝开发顺利！** 🚀

