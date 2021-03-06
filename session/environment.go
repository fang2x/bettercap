package session

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sort"
	"strconv"
	"sync"

	"github.com/bettercap/bettercap/core"
)

type SetCallback func(newValue string)

type Environment struct {
	sync.Mutex
	Data map[string]string `json:"data"`
	cbs  map[string]SetCallback
	sess *Session
}

func NewEnvironment(s *Session, envFile string) *Environment {
	env := &Environment{
		Data: make(map[string]string),
		sess: s,
		cbs:  make(map[string]SetCallback),
	}

	if envFile != "" {
		envFile, _ := core.ExpandPath(envFile)
		if core.Exists(envFile) {
			if err := env.Load(envFile); err != nil {
				fmt.Printf("Error while loading %s: %s\n", envFile, err)
			}
		}
	}

	return env
}

func (env *Environment) Load(fileName string) error {
	env.Lock()
	defer env.Unlock()

	raw, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}

	return json.Unmarshal(raw, &env.Data)
}

func (env *Environment) Save(fileName string) error {
	env.Lock()
	defer env.Unlock()

	raw, err := json.Marshal(env.Data)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(fileName, raw, 0644)
}

func (env *Environment) Has(name string) bool {
	env.Lock()
	defer env.Unlock()

	_, found := env.Data[name]

	return found
}

func (env *Environment) SetCallback(name string, cb SetCallback) {
	env.Lock()
	defer env.Unlock()
	env.cbs[name] = cb
}

func (env *Environment) WithCallback(name, value string, cb SetCallback) string {
	ret := env.Set(name, value)
	env.SetCallback(name, cb)
	return ret
}

func (env *Environment) Set(name, value string) string {
	env.Lock()
	defer env.Unlock()

	old := env.Data[name]
	env.Data[name] = value

	if cb, hasCallback := env.cbs[name]; hasCallback {
		cb(value)
	}

	env.sess.Events.Log(core.DEBUG, "env.change: %s -> '%s'", name, value)

	return old
}

func (env *Environment) Get(name string) (bool, string) {
	env.Lock()
	defer env.Unlock()

	if value, found := env.Data[name]; found {
		return true, value
	}

	return false, ""
}

func (env *Environment) GetInt(name string) (error, int) {
	if found, value := env.Get(name); found {
		if i, err := strconv.Atoi(value); err == nil {
			return nil, i
		} else {
			return err, 0
		}
	}

	return fmt.Errorf("Not found."), 0
}

func (env *Environment) Sorted() []string {
	env.Lock()
	defer env.Unlock()

	var keys []string
	for k := range env.Data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
