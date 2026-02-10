import type { RouteRecordRaw } from "vue-router"
import { routerConfig } from "@/router/config"
import { registerNavigationGuard } from "@/router/guard"
import { createRouter } from "vue-router"
import { flatMultiLevelRoutes } from "./helper"

const Layouts = () => import("@/layouts/index.vue")

/**
 * @name 常驻路由
 * @description 除了 redirect/403/404/login 等隐藏页面，其他页面建议设置唯一的 Name 属性
 */
export const constantRoutes: RouteRecordRaw[] = [
  {
    path: "/redirect",
    component: Layouts,
    meta: {
      hidden: true
    },
    children: [
      {
        path: ":path(.*)",
        component: () => import("@/pages/redirect/index.vue")
      }
    ]
  },
  {
    path: "/403",
    component: () => import("@/pages/error/403.vue"),
    meta: {
      hidden: true
    }
  },
  {
    path: "/404",
    component: () => import("@/pages/error/404.vue"),
    meta: {
      hidden: true
    },
    alias: "/:pathMatch(.*)*"
  },
  {
    path: "/login",
    component: () => import("@/pages/login/index.vue"),
    meta: {
      hidden: true
    }
  },
  {
    path: "/2fa-setup",
    component: () => import("@/pages/twofa/index.vue"),
    meta: {
      hidden: true
    }
  },
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
          title: "商户列表",
          svgIcon: "dashboard",
          affix: true
        }
      }
    ]
  },
  {
    path: "/merchant/create",
    component: Layouts,
    meta: { hidden: true },
    children: [
      {
        path: "",
        component: () => import("@/pages/dashboard/merchant-form.vue"),
        name: "MerchantCreate",
        meta: {
          title: "新增商户",
          keepAlive: false
        }
      }
    ]
  },
  {
    path: "/merchant/edit/:id",
    component: Layouts,
    meta: { hidden: true },
    children: [
      {
        path: "",
        component: () => import("@/pages/dashboard/merchant-form.vue"),
        name: "MerchantEdit",
        meta: {
          title: "编辑商户",
          keepAlive: false
        }
      }
    ]
  },
  {
    path: "/utils",
    component: Layouts,
    redirect: "/utils/tools",
    name: "Utils",
    meta: {
      title: "工具箱",
      elIcon: "Tools",
      alwaysShow: false
    },
    children: [
      {
        path: "tools",
        component: () => import("@/pages/utils/index.vue"),
        name: "UtilsTools",
        meta: {
          title: "实用工具",
          elIcon: "Tools",
          keepAlive: true
        }
      }
    ]
  },
  {
    path: "/global",
    component: Layouts,
    redirect: "/global/oss-url",
    name: "Global",
    meta: {
      title: "全局管理",
      elIcon: "Setting",
      keepAlive: true
    },
    children: [
      // {
      //   path: "oss-url",
      //   component: () => import("@/pages/global/oss-url.vue"),
      //   name: "GlobalOssUrl",
      //   meta: {
      //     title: "IP文件管理",
      //     elIcon: "Document",
      //     keepAlive: true
      //   }
      // },
      // {
      //   path: "clients",
      //   component: () => import("@/pages/clients/list.vue"),
      //   name: "GlobalClients",
      //   meta: {
      //     title: "客户端管理",
      //     elIcon: "Iphone",
      //     keepAlive: true
      //   }
      // },
      {
        path: "announcements",
        component: () => import("@/pages/announcements/index.vue"),
        name: "GlobalAnnouncements",
        meta: {
          title: "系统公告",
          elIcon: "Bell",
          keepAlive: false
        }
      },
      {
        path: "app-logs",
        component: () => import("@/pages/app-logs/index.vue"),
        name: "GlobalAppLogs",
        meta: {
          title: "应用日志",
          elIcon: "Document",
          keepAlive: false
        }
      }
    ]
  },
  {
    path: "/deploy",
    component: Layouts,
    redirect: "/deploy/servers",
    name: "Deploy",
    meta: {
      title: "运维管理",
      elIcon: "Monitor",
      keepAlive: true
    },
    children: [
      {
        path: "servers",
        component: () => import("@/pages/deploy/servers.vue"),
        name: "DeployServers",
        meta: {
          title: "服务器",
          elIcon: "Monitor",
          keepAlive: true
        }
      },
      {
        path: "servers/create",
        component: () => import("@/pages/deploy/servers-form.vue"),
        name: "DeployServersCreate",
        meta: {
          title: "新增服务器",
          hidden: true,
          keepAlive: false
        }
      },
      {
        path: "servers/edit/:id",
        component: () => import("@/pages/deploy/servers-form.vue"),
        name: "DeployServersEdit",
        meta: {
          title: "编辑服务器",
          hidden: true,
          keepAlive: false
        }
      },
      {
        path: "gost",
        component: () => import("@/pages/deploy/gost.vue"),
        name: "DeployGost",
        meta: {
          title: "隧道服务",
          elIcon: "Connection",
          keepAlive: true
        }
      },
      {
        path: "control",
        component: () => import("@/pages/deploy/control.vue"),
        name: "DeployControl",
        meta: {
          title: "部署控制",
          elIcon: "Operation",
          hidden: true,
          keepAlive: false
        }
      },
      {
        path: "docker",
        component: () => import("@/pages/deploy/docker.vue"),
        name: "DeployDocker",
        meta: {
          title: "Docker服务",
          elIcon: "Box",
          keepAlive: true,
          roles: ["admin"]
        }
      },
      {
        path: "webssh",
        component: () => import("@/pages/deploy/webssh.vue"),
        name: "DeployWebSSH",
        meta: {
          title: "SSH终端",
          elIcon: "Monitor",
          keepAlive: true,
          roles: ["admin"]
        }
      },
      {
        path: "features",
        component: () => import("@/pages/feature/index.vue"),
        name: "DeployFeatures",
        meta: {
          title: "功能开关",
          elIcon: "Switch",
          keepAlive: true
        }
      },
      {
        path: "batch",
        component: () => import("@/pages/ops/batch/index.vue"),
        name: "DeployBatch",
        meta: {
          title: "批量运维",
          elIcon: "Operation",
          keepAlive: false
        }
      },
      {
        path: "logs",
        component: () => import("@/pages/ops/logs/index.vue"),
        name: "DeployLogs",
        meta: {
          title: "日志查看",
          elIcon: "Document",
          keepAlive: false
        }
      },
      {
        path: "versions",
        component: () => import("@/pages/ops/versions/index.vue"),
        name: "DeployVersions",
        meta: {
          title: "版本管理",
          elIcon: "Files",
          keepAlive: false
        }
      },
      {
        path: "storage",
        component: () => import("@/pages/storage/merchant-storage.vue"),
        name: "DeployStorage",
        meta: {
          title: "存储配置",
          elIcon: "FolderOpened",
          keepAlive: false
        }
      }
    ]
  },
  {
    path: "/deploy/webssh-full",
    component: () => import("@/pages/deploy/webssh-full.vue"),
    name: "DeployWebSSHFull",
    meta: {
      title: "SSH全屏",
      hidden: true,
      keepAlive: false,
      roles: ["admin"]
    }
  },
  {
    path: "/cloud",
    component: Layouts,
    redirect: "/cloud/aliyun",
    name: "Cloud",
    meta: {
      title: "云管理",
      elIcon: "MostlyCloudy",
      keepAlive: true
    },
    children: [
      {
        path: "cloud_account",
        component: () => import("@/pages/cloud/cloud_account/index.vue"),
        name: "CloudAccount",
        meta: {
          title: "云账号管理",
          elIcon: "Key",
          keepAlive: true
        }
      },
      {
        path: "aliyun",
        name: "CloudAliyun",
        meta: {
          title: "阿里云",
          elIcon: "MostlyCloudy",
          keepAlive: true
        },
        children: [
          {
            path: "instances",
            component: () => import("@/pages/cloud/aliyun/instances/index.vue"),
            name: "CloudInstances",
            meta: {
              title: "实例列表",
              elIcon: "Connection",
              keepAlive: true
            }
          },
          {
            path: "instances/create",
            component: () => import("@/pages/cloud/aliyun/instances/create.vue"),
            name: "CloudInstancesCreate",
            meta: {
              title: "创建实例",
              hidden: true,
              keepAlive: true
            }
          },
          {
            path: "eip",
            component: () => import("@/pages/cloud/aliyun/eip/index.vue"),
            name: "CloudEIP",
            meta: {
              title: "动态IP",
              elIcon: "Location",
              keepAlive: true
            }
          },
          {
            path: "eip/create",
            component: () => import("@/pages/cloud/aliyun/eip/create.vue"),
            name: "CloudEIPCreate",
            meta: {
              title: "创建弹性IP",
              hidden: true,
              keepAlive: true
            }
          },
          {
            path: "network",
            component: () => import("@/pages/cloud/aliyun/network/index.vue"),
            name: "CloudNetwork",
            meta: {
              title: "网卡列表",
              elIcon: "Cpu",
              keepAlive: true
            }
          },
          {
            path: "bandwidth",
            component: () => import("@/pages/cloud/aliyun/bandwidth/index.vue"),
            name: "CloudBandwidth",
            meta: {
              title: "共享带宽",
              elIcon: "Operation",
              keepAlive: true
            }
          },
          {
            path: "securitygroup",
            component: () => import("@/pages/cloud/aliyun/securitygroup/index.vue"),
            name: "CloudSecurityGroup",
            meta: {
              title: "安全组",
              elIcon: "Lock",
              keepAlive: true,
              roles: ["admin"]
            }
          },
          {
            path: "securitygroup/create",
            component: () => import("@/pages/cloud/aliyun/securitygroup/create.vue"),
            name: "CloudSecurityGroupCreate",
            meta: {
              title: "创建安全组",
              hidden: true,
              keepAlive: true,
              roles: ["admin"]
            }
          },
          {
            path: "oss",
            component: () => import("@/pages/cloud/aliyun/oss/index.vue"),
            name: "CloudAliyunOSS",
            meta: {
              title: "对象存储",
              elIcon: "Files",
              keepAlive: true
            }
          },
          {
            path: "images",
            component: () => import("@/pages/cloud/aliyun/images/index.vue"),
            name: "CloudAliyunImages",
            meta: {
              title: "镜像管理",
              elIcon: "PictureFilled",
              keepAlive: true
            }
          }
        ]
      },
      {
        path: "aws",
        name: "CloudAWS",
        meta: {
          title: "AWS",
          elIcon: "MostlyCloudy",
          keepAlive: true
        },
        children: [
          {
            path: "instances",
            component: () => import("@/pages/cloud/aws/instances.vue"),
            name: "AwsInstances",
            meta: { title: "实例列表", elIcon: "Cpu", keepAlive: true }
          },
          {
            path: "instances/create",
            component: () => import("@/pages/cloud/aws/instances-create.vue"),
            name: "AwsInstancesCreate",
            meta: { title: "创建实例", hidden: true, keepAlive: true }
          },
          {
            path: "security-group",
            component: () => import("@/pages/cloud/aws/security-group.vue"),
            name: "AwsSecurityGroup",
            meta: { title: "安全组", elIcon: "Lock", keepAlive: true, roles: ["admin"] }
          },
          {
            path: "eip",
            component: () => import("@/pages/cloud/aws/eip.vue"),
            name: "AwsEip",
            meta: { title: "动态IP", elIcon: "Location", keepAlive: true }
          },
          {
            path: "index",
            component: () => import("@/pages/cloud/aws/storage.vue"),
            name: "CloudAWSIndex",
            meta: {
              title: "对象存储",
              elIcon: "Files",
              keepAlive: true
            }
          }
        ]
      },
      {
        path: "tencent",
        name: "CloudTencent",
        meta: {
          title: "腾讯云",
          elIcon: "MostlyCloudy",
          keepAlive: true
        },
        children: [
          {
            path: "instances",
            component: () => import("@/pages/cloud/tencent/instances/index.vue"),
            name: "TencentInstances",
            meta: {
              title: "实例列表",
              elIcon: "Cpu",
              keepAlive: true
            }
          },
          {
            path: "cos",
            component: () => import("@/pages/cloud/tencent/cos/index.vue"),
            name: "CloudTencentCOS",
            meta: {
              title: "对象存储",
              elIcon: "Files",
              keepAlive: true
            }
          }
        ]
      }
    ]
  }
]

/**
 * @name 动态路由
 * @description 用来放置有权限 (Roles 属性) 的路由
 * @description 必须带有唯一的 Name 属性
 */
export const dynamicRoutes: RouteRecordRaw[] = [
  // {
  //   path: "/permission",
  //   component: Layouts,
  //   redirect: "/permission/page-level",
  //   name: "Permission",
  //   meta: {
  //     title: "权限演示",
  //     elIcon: "Lock",
  //     // 可以在根路由中设置角色
  //     roles: ["admin", "editor"],
  //     alwaysShow: true
  //   },
  //   children: [
  //     {
  //       path: "page-level",
  //       component: () => import("@/pages/demo/permission/page-level.vue"),
  //       name: "PermissionPageLevel",
  //       meta: {
  //         title: "页面级",
  //         // 或者在子路由中设置角色
  //         roles: ["admin"]
  //       }
  //     },
  //     {
  //       path: "button-level",
  //       component: () => import("@/pages/demo/permission/button-level.vue"),
  //       name: "PermissionButtonLevel",
  //       meta: {
  //         title: "按钮级",
  //         // 如果未设置角色，则表示：该页面不需要权限，但会继承根路由的角色
  //         roles: undefined
  //       }
  //     }
  //   ]
  // }
]

/** 路由实例 */
export const router = createRouter({
  history: routerConfig.history,
  routes: routerConfig.thirdLevelRouteCache ? flatMultiLevelRoutes(constantRoutes) : constantRoutes
})

/** 重置路由 */
export function resetRouter() {
  try {
    // 注意：所有动态路由路由必须带有 Name 属性，否则可能会不能完全重置干净
    router.getRoutes().forEach((route) => {
      const { name, meta } = route
      if (name && meta.roles?.length) {
        router.hasRoute(name) && router.removeRoute(name)
      }
    })
  } catch {
    // 强制刷新浏览器也行，只是交互体验不是很好
    location.reload()
  }
}

// 注册路由导航守卫
registerNavigationGuard(router)
