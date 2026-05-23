package configs

import ( 
	"fmt"
	"os"
	"path/filepath"
	"github.com/BurntSushi/toml"
)


type Config struct {
	Name            string
	Version         string
	Desc            string
	DefaultProvider string   `toml:"default_provider"`
	Provider        Provider `toml:"provider"`
}

type Provider struct {
	BaiLian  LLMConfig `toml:"BaiLian"`
	DeepSeek LLMConfig `toml:"DeepSeek"`
	GPT      LLMConfig `toml:"GPT"`
}

type LLMConfig struct {
	Model string
	ApiKey string 
}



/**
 * 获取配置文件路径的方法
 */
func GetConfigPath() string{
	home, _ := os.UserHomeDir()

	return filepath.Join(home, "./seeed-cli", "config.toml")
}


// defaultConfig 首次运行时写入用户目录的默认 TOML 内容
func defaultConfig() Config {
	return Config{
		Name:            "seeed-cli",
		Version:         "0.0.1",
		Desc:            "源于 AI，归于 AI，所有输出均由 AI 生成，建议将安全或者质量评测结果再次交给您的 AI 来处理。",
		DefaultProvider: "BaiLian",
		Provider: Provider{
			BaiLian: LLMConfig{
				Model:  "qwen3.6-flash",
				ApiKey: "",
			},
			DeepSeek: LLMConfig{
				Model:  "deepseek-v4-pro",
				ApiKey: "",
			},
			GPT: LLMConfig{
				Model:  "gpt-4o",
				ApiKey: "",
			},
		},
	}
}

/**
 * 初始化配置文件
 */

func Init() error {
	path := GetConfigPath()

	// 不存在文件时需要创建
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		cfg := defaultConfig()
		return SaveConfig(path, &cfg)
	}
	return nil
}



/**
 * 保存文件
 */

func SaveConfig(path string, cfg *Config) error{
	file, err := os.Create(path)
	if err != nil{
		return err
	}

	defer file.Close()

	return toml.NewEncoder(file).Encode(cfg)
}

/**
 * 加载配置文件
 */
func LoadConfig() (*Config, error){ 
	path := GetConfigPath() 
	var cfg Config
	_, err := toml.DecodeFile(path, &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}


/**
 * 设置 api-key
 */
func SetAK(provider string, ak string) error { 
	path := GetConfigPath() 
	var cfg Config
	_, err := toml.DecodeFile(path, &cfg)

	if err != nil{
		return err
	}

	switch provider {
	case "BaiLian":
		cfg.Provider.BaiLian.ApiKey = ak
	case "DeepSeek":
		cfg.Provider.DeepSeek.ApiKey = ak
	case "GPT":
		cfg.Provider.GPT.ApiKey = ak
	default:
		return fmt.Errorf("unknown provider: %s", provider)
	}

	SaveConfig(path, &cfg)
	return nil
}


/**
 * 获取 api-key
 */
func GetAK(provider string) (string, error) { 
	path := GetConfigPath() 

	if _, err := os.Stat(path); os.IsNotExist(err){
		return "", err
	}

	var cfg Config
	_, err := toml.DecodeFile(path, &cfg)

	if err != nil{
		return  "", err
	}
	
	switch provider {
	case "BaiLian":
		return cfg.Provider.BaiLian.ApiKey, nil
	case "DeepSeek":
		return cfg.Provider.DeepSeek.ApiKey, nil
	case "GPT":
		return cfg.Provider.GPT.ApiKey, nil
	default:
		return "", fmt.Errorf("unsupported provider: %s", provider)
	}
}

// GetProviderBaseURL 返回各 provider 的 OpenAI 兼容端点
func GetProviderBaseURL(provider string) string {
	switch provider {
	case "BaiLian":
		return "https://dashscope.aliyuncs.com/compatible-mode/v1"
	case "DeepSeek":
		return "https://api.deepseek.com/v1"
	case "GPT":
		return "https://api.openai.com/v1"
	default:
		return ""
	}
}

// GetProviderModel 从配置获取 provider 对应的模型名，模型为空时返回内置默认值
func GetProviderModel(cfg *Config, provider string) string {
	switch provider {
	case "BaiLian":
		if cfg.Provider.BaiLian.Model != "" {
			return cfg.Provider.BaiLian.Model
		}
		return "qwen3.6-flash"
	case "DeepSeek":
		if cfg.Provider.DeepSeek.Model != "" {
			return cfg.Provider.DeepSeek.Model
		}
		return "deepseek-chat"
	case "GPT":
		if cfg.Provider.GPT.Model != "" {
			return cfg.Provider.GPT.Model
		}
		return "gpt-4o"
	default:
		return ""
	}
}

