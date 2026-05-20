// assets：使用 go:embed 把 skills 索引清单等静态资源内嵌到二进制中。
package assets

import _ "embed"

// TopYAML：skills-pull 命令使用的 Top-50 静态种子清单（YAML 格式）。
//
//go:embed skills-top50.yaml
var TopYAML []byte
