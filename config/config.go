package config

import (
	"bufio"
	"go-redis/lib/logger"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"
)

// 解析配置(redis.conf)
// ServerProperties defines global config properties
type ServerProperties struct {
	Bind           string `cfg:"bind"`           //监听的IP
	Port           int    `cfg:"port"`           //监听的端口
	AppendOnly     bool   `cfg:"appendOnly"`     //Redis的持久化存储提供两种方式：RDB与AOF。RDB是默认配置，AOF需要手动开启
	AppendFilename string `cfg:"appendFilename"` //保存数据的AOF文件名称
	MaxClients     int    `cfg:"maxclients"`     //redis允许的最大连接数
	RequirePass    string `cfg:"requirepass"`    //登录密码
	Databases      int    `cfg:"databases"`      //设置redis中数据库的总数

	Peers []string `cfg:"peers"`
	Self  string   `cfg:"self"`
}

// Properties holds global config properties
var Properties *ServerProperties

func init() {
	// default config
	Properties = &ServerProperties{
		Bind:       "127.0.0.1",
		Port:       6379,
		AppendOnly: false,
	}
}

// 解析配置文件函数
func parse(src io.Reader) *ServerProperties {
	config := &ServerProperties{}

	// read config file
	rawMap := make(map[string]string)
	scanner := bufio.NewScanner(src) //Scanner是一个结构体
	for scanner.Scan() {             //每次调用 scanner.Scan()，即 读入下一行 ，并移除行末的换行符。
		line := scanner.Text() //读取的内容可以调用 scanner.Text() 得到。Scan函数在读到一行时返回true，不再有输入时返回false。
		if len(line) > 0 && line[0] == '#' {
			continue
		}
		//strings.IndexAny()返回字符串str中的任何一个字符在字符串s中第一次出现的位置。
		//如果找不到或str为空则返回-1
		//以下代码效果：读取配置文件一行 bind 0.0.0.0 => map["bind"] = "0.0.0.0"
		pivot := strings.IndexAny(line, " ")
		if pivot > 0 && pivot < len(line)-1 { // separator found
			key := line[0:pivot]
			value := strings.Trim(line[pivot+1:], " ")
			rawMap[strings.ToLower(key)] = value
		}
	}
	if err := scanner.Err(); err != nil {
		logger.Fatal(err)
	}

	// 通过反射，把值填充给结构体，因版本问题，不使用第三方配置文件解析
	// parse format
	t := reflect.TypeOf(config)
	v := reflect.ValueOf(config)
	n := t.Elem().NumField()
	for i := 0; i < n; i++ {
		field := t.Elem().Field(i)
		fieldVal := v.Elem().Field(i)
		key, ok := field.Tag.Lookup("cfg")
		if !ok {
			key = field.Name
		}
		value, ok := rawMap[strings.ToLower(key)]
		if ok {
			// fill config
			switch field.Type.Kind() {
			case reflect.String:
				fieldVal.SetString(value)
			case reflect.Int:
				intValue, err := strconv.ParseInt(value, 10, 64)
				if err == nil {
					fieldVal.SetInt(intValue)
				}
			case reflect.Bool:
				boolValue := "yes" == value
				fieldVal.SetBool(boolValue)
			case reflect.Slice:
				if field.Type.Elem().Kind() == reflect.String {
					slice := strings.Split(value, ",")
					fieldVal.Set(reflect.ValueOf(slice))
				}
			}
		}
	}

	return config
}

// SetupConfig read config file and store properties into Properties
func SetupConfig(configFilename string) {
	file, err := os.Open(configFilename)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	//解析配置文件
	Properties = parse(file)
}
