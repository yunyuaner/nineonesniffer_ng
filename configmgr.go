package nineonesniffer

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

const (
	defaultConfigFile = "NineOneSniffer.conf"
)

var defaultNineOneConfig map[string]string

type nineOneSnifferConfig struct {
	workDir               string
	baseURL               string
	listPageURLBase       string
	videoPartsURLBase     string
	userAgent             string
	configBaseDir         string
	cookieFile            string
	dataBaseDir           string
	videoPartsDir         string
	videoMergedDir        string
	videoPartsDescTodoDir string
	videoPartsDescDoneDir string
	videoListBaseDir      string
	thumbnailBaseDir      string
	thumbnailNewDir       string
	utilsDir              string
	sqliteDir             string
	tempDir               string
}

type NineOneConfManager struct {
	config    nineOneSnifferConfig
	configMap map[string]string
}

func (confmgr *NineOneConfManager) Start(configFile string) error {
	confmgr.configMap = make(map[string]string)

	if err := confmgr.parseConfig(configFile); err != nil {
		fmt.Println(err)
		return err
	}

	confmgr.makeConfigStruct()
	// confmgr.showConfig()

	return nil
}

func (confmgr *NineOneConfManager) preParseRtn(line string) (lineNew string, ignore bool) {
	/* trim leading and trailing spaces */
	if lineNew = strings.Trim(line, " \r"); len(lineNew) == 0 {
		return lineNew, true
	}

	/* ignore comment line */
	re := regexp.MustCompile(`^#.*`)
	if matched := re.Match([]byte(lineNew)); matched {
		return lineNew, true
	}

	/* check config line pattern */
	re = regexp.MustCompile(`[a-zA-Z0-9_\s]*=[a-zA-Z0-9_\s\$\{\}]*`)
	if matched := re.Match([]byte(lineNew)); !matched {
		fmt.Printf("invalid config value pattern - %s\n", lineNew)
		return lineNew, true
	}

	return lineNew, false
}

func (confmgr *NineOneConfManager) parsePatternRtn(line string) (configName, configValue string) {
	var configWithoutValue bool

	tokenPos := strings.Index(line, "=")
	configName = strings.TrimRight(line[:tokenPos], " \r")
	if !configWithoutValue {
		configValue = strings.TrimLeft(line[tokenPos+1:], " \r")
	}

	/* de-Quote */
	configValue = strings.Trim(configValue, "\"'")

	return configName, configValue
}

func (confmgr *NineOneConfManager) postParseRtn(configName string, info interface{}) (string, string, error) {
	item := info.(*configValueWithVariable)
	variables := item.variables

	/* For each matched variables, expand variable with it's value */
	for _, subs := range variables {
		subs = subs[2 : len(subs)-1]
		/* find variable name in config map */
		if _, ok := confmgr.configMap[subs]; ok {
			variableValue := confmgr.configMap[subs]
			configValue := strings.Replace(item.configValue, "${"+subs+"}", variableValue, -1)
			confmgr.configMap[configName] = configValue
		} else {
			/* unrecognized variable found, stop and return with error */
			return configName, item.configValue, fmt.Errorf("unrecognized variable found")
		}
	}

	// fmt.Printf("configName - %s, Expended - %s\n", configName, confmgr.configMap[configName])

	return configName, confmgr.configMap[configName], nil
}

type configValueWithVariable struct {
	configValue string
	variables   []string
	done        bool
}

func (confmgr *NineOneConfManager) parseConfig(configFile string) error {
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return err
	}

	file, err := os.Open(configFile)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	configToBeExpandedMap := make(map[string]*configValueWithVariable)

	for scanner.Scan() {
		line := scanner.Text()
		line, ignore := confmgr.preParseRtn(line)
		if ignore {
			continue
		}

		configName, configValue := confmgr.parsePatternRtn(line)

		re := regexp.MustCompile(`\$\{[a-zA-Z0-9_]+}`)
		subMatch := re.FindStringSubmatch(configValue)
		if len(subMatch) > 0 {
			configToBeExpandedMap[configName] = &configValueWithVariable{configValue, subMatch, false}
		} else {
			confmgr.configMap[configName] = configValue
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	if _, ok := confmgr.configMap["work_dir"]; !ok || len(confmgr.configMap["work_dir"]) == 0 {
		confmgr.configMap["work_dir"], _ = os.Getwd()
	}

	/* expand variables if any */

	for {
		for configName, items := range configToBeExpandedMap {
			configName, expandedValue, err := confmgr.postParseRtn(configName, items)
			if err != nil {
				continue
			} else {
				items.done = true
				delete(configToBeExpandedMap, configName)
			}

			confmgr.configMap[configName] = expandedValue
		}

		if len(configToBeExpandedMap) == 0 {
			break
		}
	}

	return nil
}

func (confmgr *NineOneConfManager) makeConfigStruct() {
	defaultNineOneConfig := map[string]string{
		"base_url":                  "http://www.91porn.com",
		"alt_base_url":              "https://e1015.91p01.com",
		"list_page_base_url":        "${bass_url}/v.php?next=watch&page=",
		"video_parts_url_base":      "https://cdn.91p07.com//m3u8",
		"user_agent":                "Mozilla/5.0 (platform; rv:17.0) Gecko/20100101 SeaMonkey/2.7.1",
		"work_dir":                  "./",
		"config_base_dir":           "${work_dir}configs",
		"cookie_file":               "${config_base_dir}/cookies.txt",
		"data_base_dir":             "${work_dir}/data",
		"video_parts_dir":           "${data_base_dir}/video/video_parts",
		"vidro_merged_dir":          "${data_base_dir}/video/video_merged",
		"video_parts_desc_todo_dir": "${data_base_dir}/video/m3u8/todo",
		"video_parts_desc_done_dir": "${data_base_dir}/video/m3u8/done",
		"video_list_base_dir":       "${data_base_dir}/list",
		"thumbnail_base_dir":        "${data_base_dir}/images/base",
		"thumbnail_new_dir":         "${data_base_dir}/images/new",
		"utils_dir":                 "${work_dir}/utils",
		"sqlite_dir":                "${data_base_dir}",
		"temp_dir":                  "${work_dir}/tmp",
	}

	configMapVal := func(configName string) string {
		if value, ok := confmgr.configMap[configName]; !ok {
			return defaultNineOneConfig[configName]
		} else {
			return value
		}
	}

	confmgr.config.workDir = configMapVal("work_dir")
	confmgr.config.baseURL = configMapVal("base_url")
	confmgr.config.listPageURLBase = configMapVal("list_page_base_url")
	confmgr.config.videoPartsURLBase = configMapVal("video_parts_url_base")
	confmgr.config.userAgent = configMapVal("user_agent")
	confmgr.config.configBaseDir = configMapVal("config_base_dir")
	confmgr.config.cookieFile = configMapVal("cookie_file")
	confmgr.config.dataBaseDir = configMapVal("data_base_dir")
	confmgr.config.videoPartsDir = configMapVal("video_parts_dir")
	confmgr.config.videoMergedDir = configMapVal("video_merged_dir")
	confmgr.config.videoPartsDescTodoDir = configMapVal("video_parts_desc_todo_dir")
	confmgr.config.videoPartsDescDoneDir = configMapVal("video_parts_desc_done_dir")
	confmgr.config.videoListBaseDir = configMapVal("video_list_base_dir")
	confmgr.config.thumbnailBaseDir = configMapVal("thumbnail_base_dir")
	confmgr.config.thumbnailNewDir = configMapVal("thumbnail_new_dir")
	confmgr.config.utilsDir = configMapVal("utils_dir")
	confmgr.config.sqliteDir = configMapVal("sqlite_dir")
	confmgr.config.tempDir = configMapVal("temp_dir")
}

func (confmgr *NineOneConfManager) showConfig() {
	fmt.Printf("baseURL - %s\n", confmgr.config.baseURL)
	fmt.Printf("videoPartsURLBase - %s\n", confmgr.config.videoPartsURLBase)
	fmt.Printf("userAgent - %s\n", confmgr.config.userAgent)
	fmt.Printf("configBaseDir - %s\n", confmgr.config.configBaseDir)
	fmt.Printf("cookieFile - %s\n", confmgr.config.cookieFile)
	fmt.Printf("dataBaseDir - %s\n", confmgr.config.dataBaseDir)
	fmt.Printf("videoPartsDir - %s\n", confmgr.config.videoPartsDir)
	fmt.Printf("videoMergedDir - %s\n", confmgr.config.videoMergedDir)
	fmt.Printf("videoPartsDescTodoDir - %s\n", confmgr.config.videoPartsDescTodoDir)
	fmt.Printf("videoPartsDescDoneDir - %s\n", confmgr.config.videoPartsDescDoneDir)
	fmt.Printf("videoListBaseDir - %s\n", confmgr.config.videoListBaseDir)
	fmt.Printf("thumbnailBaseDir - %s\n", confmgr.config.thumbnailBaseDir)
	fmt.Printf("thumbnailNewDir - %s\n", confmgr.config.thumbnailNewDir)
	fmt.Printf("utilsDir - %s\n", confmgr.config.utilsDir)
	fmt.Printf("sqliteDir - %s\n", confmgr.config.sqliteDir)
	fmt.Printf("tempDir - %s\n", confmgr.config.tempDir)
}
