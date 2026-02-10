export interface AwsRegionMeta {
  id: string
  nameEn: string
  nameCn: string
}

const AWS_REGIONS_META: AwsRegionMeta[] = [
  { id: "ap-southeast-1", nameEn: "Asia Pacific (Singapore)", nameCn: "亚太（新加坡）" },
  { id: "ap-east-1", nameEn: "Asia Pacific (Hong Kong)", nameCn: "亚太（香港）" },
  { id: "ap-south-2", nameEn: "Asia Pacific (Hyderabad)", nameCn: "亚太（海得拉巴）" },
  { id: "ap-southeast-3", nameEn: "Asia Pacific (Jakarta)", nameCn: "亚太（雅加达）" },
  { id: "ap-southeast-5", nameEn: "Asia Pacific (Malaysia)", nameCn: "亚太（马来西亚）" },
  { id: "ap-southeast-4", nameEn: "Asia Pacific (Melbourne)", nameCn: "亚太（墨尔本）" },
  { id: "ap-south-1", nameEn: "Asia Pacific (Mumbai)", nameCn: "亚太（孟买）" },
  { id: "ap-southeast-6", nameEn: "Asia Pacific (New Zealand)", nameCn: "亚太（新西兰）" },
  { id: "ap-northeast-3", nameEn: "Asia Pacific (Osaka)", nameCn: "亚太（大阪）" },
  { id: "ap-northeast-2", nameEn: "Asia Pacific (Seoul)", nameCn: "亚太（首尔）" },
  { id: "ap-southeast-2", nameEn: "Asia Pacific (Sydney)", nameCn: "亚太（悉尼）" },
  { id: "ap-east-2", nameEn: "Asia Pacific (Taipei)", nameCn: "亚太（台北）" },
  { id: "ap-southeast-7", nameEn: "Asia Pacific (Thailand)", nameCn: "亚太（泰国）" },
  { id: "ap-northeast-1", nameEn: "Asia Pacific (Tokyo)", nameCn: "亚太（东京）" },
  { id: "us-east-1", nameEn: "US East (N. Virginia)", nameCn: "美国东部（弗吉尼亚北部）" },
  { id: "us-east-2", nameEn: "US East (Ohio)", nameCn: "美国东部（俄亥俄）" },
  { id: "us-west-1", nameEn: "US West (N. California)", nameCn: "美国西部（加利福尼亚北部）" },
  { id: "us-west-2", nameEn: "US West (Oregon)", nameCn: "美国西部（俄勒冈）" },
  { id: "af-south-1", nameEn: "Africa (Cape Town)", nameCn: "非洲（开普敦）" },
  { id: "ca-central-1", nameEn: "Canada (Central)", nameCn: "加拿大（中部）" },
  { id: "ca-west-1", nameEn: "Canada West (Calgary)", nameCn: "加拿大西部（卡尔加里）" },
  { id: "eu-central-1", nameEn: "Europe (Frankfurt)", nameCn: "欧洲（法兰克福）" },
  { id: "eu-west-1", nameEn: "Europe (Ireland)", nameCn: "欧洲（爱尔兰）" },
  { id: "eu-west-2", nameEn: "Europe (London)", nameCn: "欧洲（伦敦）" },
  { id: "eu-south-1", nameEn: "Europe (Milan)", nameCn: "欧洲（米兰）" },
  { id: "eu-west-3", nameEn: "Europe (Paris)", nameCn: "欧洲（巴黎）" },
  { id: "eu-south-2", nameEn: "Europe (Spain)", nameCn: "欧洲（西班牙）" },
  { id: "eu-north-1", nameEn: "Europe (Stockholm)", nameCn: "欧洲（斯德哥尔摩）" },
  { id: "eu-central-2", nameEn: "Europe (Zurich)", nameCn: "欧洲（苏黎世）" },
  { id: "il-central-1", nameEn: "Israel (Tel Aviv)", nameCn: "以色列（特拉维夫）" },
  { id: "mx-central-1", nameEn: "Mexico (Central)", nameCn: "墨西哥（中部）" },
  { id: "me-south-1", nameEn: "Middle East (Bahrain)", nameCn: "中东（巴林）" },
  { id: "me-central-1", nameEn: "Middle East (UAE)", nameCn: "中东（阿联酋）" },
  { id: "sa-east-1", nameEn: "South America (São Paulo)", nameCn: "南美（圣保罗）" }
]

export interface RegionOption { id: string, name: string }

export function getAwsRegions(lang: "cn" | "en" = "cn"): RegionOption[] {
  return AWS_REGIONS_META.map(r => ({ id: r.id, name: lang === "en" ? r.nameEn : r.nameCn }))
}
