package space

import (
	"slack-wails/lib/structs"
	"strings"
)

// 处理空间引擎的位置数据，删除非空空项，并处理直辖市只显示一次，按照连接符返回字符串
func MergePosition(position structs.Position) string {
	fields := []string{}
	if position.Country != "" {
		fields = append(fields, position.Country)
	}
	if position.Province != "" && position.Province != position.City {
		fields = append(fields, position.Province)
	}
	if position.City != "" {
		fields = append(fields, position.City)
	}
	if position.District != "" {
		fields = append(fields, position.District)
	}
	return strings.Join(fields, position.Connector)
}
