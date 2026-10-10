export function jobErrorHelp(error: string): string {
  const text = error.toLowerCase();
  if (/changed|fingerprint|已变化|重新导入|re-import/.test(text))
    return "源文件已变化，请重新导入后再处理。";
  if (/already exists|file exists|已存在/.test(text))
    return "输出文件已存在，请选择新的文件名。";
  if (/permission|access is denied|拒绝访问/.test(text))
    return "无法写入输出位置，请检查权限或换一个文件夹。";
  if (/no space|disk full|空间不足/.test(text))
    return "磁盘空间不足，请腾出空间后重新处理。";
  if (/encoder|nvenc|qsv|amf/.test(text))
    return "编码器执行失败，可以尝试选择 CPU 或其他编码格式，再检查详细日志。";
  return "处理未完成，请查看详细原因。重新执行前会再次检查源文件与输出设置。";
}
