import * as XLSX from "xlsx"

/**
 * 导出数据为Excel文件
 * @param data 要导出的数据数组
 * @param filename 文件名（不含扩展名）
 * @param sheetName 工作表名称
 */
export function exportToExcel(data: any[], filename: string, sheetName = "Sheet1") {
  // 创建工作簿
  const wb = XLSX.utils.book_new()

  // 将数据转换为工作表
  const ws = XLSX.utils.json_to_sheet(data)

  // 添加工作表到工作簿
  XLSX.utils.book_append_sheet(wb, ws, sheetName)

  // 导出文件
  XLSX.writeFile(wb, `${filename}.xlsx`)
}

/**
 * 导出数据为CSV文件
 * @param data 要导出的数据数组
 * @param filename 文件名（不含扩展名）
 */
export function exportToCSV(data: any[], filename: string) {
  // 创建工作簿
  const wb = XLSX.utils.book_new()

  // 将数据转换为工作表
  const ws = XLSX.utils.json_to_sheet(data)

  // 添加工作表到工作簿
  XLSX.utils.book_append_sheet(wb, ws, "Sheet1")

  // 导出为CSV文件
  XLSX.writeFile(wb, `${filename}.csv`, { bookType: "csv" })
}

/**
 * 导出数据（自动根据格式选择）
 * @param data 要导出的数据数组
 * @param filename 文件名（不含扩展名）
 * @param format 导出格式：excel 或 csv
 */
export function exportData(data: any[], filename: string, format: "excel" | "csv" = "excel") {
  if (format === "csv") {
    exportToCSV(data, filename)
  } else {
    exportToExcel(data, filename)
  }
}
