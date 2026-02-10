/**
 * 腾讯云 COS 地区常量配置
 * https://cloud.tencent.com/document/product/436/6224
 */
export interface TencentRegionMeta {
  id: string
  nameEn: string
  nameCn: string
}

const TENCENT_REGIONS_META: TencentRegionMeta[] = [
  // 中国大陆地域
  { id: "ap-beijing-1", nameEn: "Beijing Zone 1", nameCn: "北京一区" },
  { id: "ap-beijing", nameEn: "Beijing", nameCn: "北京" },
  { id: "ap-nanjing", nameEn: "Nanjing", nameCn: "南京" },
  { id: "ap-shanghai", nameEn: "Shanghai", nameCn: "上海" },
  { id: "ap-guangzhou", nameEn: "Guangzhou", nameCn: "广州" },
  { id: "ap-chengdu", nameEn: "Chengdu", nameCn: "成都" },
  { id: "ap-chongqing", nameEn: "Chongqing", nameCn: "重庆" },
  { id: "ap-shenzhen-fsi", nameEn: "Shenzhen Finance", nameCn: "深圳金融" },
  { id: "ap-shanghai-fsi", nameEn: "Shanghai Finance", nameCn: "上海金融" },
  { id: "ap-beijing-fsi", nameEn: "Beijing Finance", nameCn: "北京金融" },
  // 中国香港及境外地域
  { id: "ap-hongkong", nameEn: "Hong Kong (China)", nameCn: "中国香港" },
  { id: "ap-singapore", nameEn: "Singapore", nameCn: "新加坡" },
  { id: "ap-mumbai", nameEn: "Mumbai", nameCn: "孟买" },
  { id: "ap-jakarta", nameEn: "Jakarta", nameCn: "雅加达" },
  { id: "ap-seoul", nameEn: "Seoul", nameCn: "首尔" },
  { id: "ap-bangkok", nameEn: "Bangkok", nameCn: "曼谷" },
  { id: "ap-tokyo", nameEn: "Tokyo", nameCn: "东京" },
  { id: "na-siliconvalley", nameEn: "Silicon Valley", nameCn: "硅谷" },
  { id: "na-ashburn", nameEn: "Virginia", nameCn: "弗吉尼亚" },
  { id: "na-toronto", nameEn: "Toronto", nameCn: "多伦多" },
  { id: "sa-saopaulo", nameEn: "São Paulo", nameCn: "圣保罗" },
  { id: "eu-frankfurt", nameEn: "Frankfurt", nameCn: "法兰克福" }
]

export interface RegionOption {
  id: string
  name: string
}

export function getTencentRegions(lang: "cn" | "en" = "cn"): RegionOption[] {
  return TENCENT_REGIONS_META.map(r => ({
    id: r.id,
    name: lang === "en" ? r.nameEn : r.nameCn
  }))
}

export function getTencentRegionName(regionId: string, lang: "cn" | "en" = "cn"): string {
  const region = TENCENT_REGIONS_META.find(r => r.id === regionId)
  if (!region) return regionId
  return lang === "en" ? region.nameEn : region.nameCn
}
