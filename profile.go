package dollarYaml

import (
	"errors"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
)

type YamlProfile map[interface{}]interface{}

func (this *YamlProfile) Read(data []byte) *YamlProfile {
	yaml.Unmarshal(data, this)
	return this
}
func (this *YamlProfile) ReadFromPath(path string) *YamlProfile {
	bb, _ := ioutil.ReadFile(path)
	yaml.Unmarshal(bb, this)
	return this
}
func (this YamlProfile) Get(path string) string {
	_, result := this.GetError(path)
	return result
}

func (this YamlProfile) GetError(path string) (error, string) {
	er, result := this.get(path)
	return er, result
}

func (this YamlProfile) get(path string) (error, string) {
	paths := strings.Split(path, ".")
	step := len(paths)
	thiz := this

	for i, p := range paths {
		d, ok := thiz[p]
		if ok {
			tp := reflect.TypeOf(d)
			if tp.Kind() == reflect.Map {
				if step == i+1 {
					return errors.New("can't find value"), ""
				} else {
					thiz = d.(YamlProfile)
				}

			} else {
				var result = ""
				var err error = nil
				if step != i+1 {
					err = errors.New("level does not match")
				}
				if tp.Kind() == reflect.String {
					result = d.(string)
					if strings.Index(result, "${") == 0 && strings.Index(result, "}") == len(result)-1 {
						result = strings.ReplaceAll(result, "${", "")
						result = strings.ReplaceAll(result, "}", "")
						//判断为包含环境变量模式,
						departs := strings.Index(result, ":")
						envName := result[:departs]
						envValue := os.Getenv(envName)
						if len(envValue) > 0 {
							result = envValue
						} else {
							result = result[departs+1:]
						}
					}
				} else {
					result = fmt.Sprint(d)
				}
				return err, result
			}
		} else {
			return errors.New(fmt.Sprintf("can't find value of '%v'", p)), ""
		}
	}
	return errors.New("can't find value"), ""
}
