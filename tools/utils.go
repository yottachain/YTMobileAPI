package tools

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"strings"
)

type User struct {
	UserName   string
	Num        uint32
	PrivateKey string
	PublicKey  string
}

func ReadUserInfo() []byte {
	fp, err := os.OpenFile("./user.json", os.O_RDONLY, 0755)
	defer fp.Close()
	if err != nil {
		logrus.Errorf("err:%s\n", err)
	}
	data := make([]byte, 1024)
	n, err := fp.Read(data)
	if err != nil {
		logrus.Errorf("err:%s\n", err)
	}
	//fmt.Println(string(data[:n]))
	return data[:n]
}

func UserUnmarshal(data []byte) User {
	var user User
	if len(data) == 0 {
		fmt.Println("User JSON is null....")
	}
	err := json.Unmarshal(data, &user)
	if err != nil {
		logrus.Errorf("err:%s\n", err)
	}
	//fmt.Println("UserName:" + user.UserName)
	//fmt.Println("PrivateKey:" + user.PrivateKey)
	//fmt.Println("PublicKey:" + user.PublicKey)
	return user
}

func CreateDirectory(directory, fileName string) string {
	s, err := os.Stat(directory)
	if err != nil {
		if !os.IsExist(err) {
			err = os.MkdirAll(directory, os.ModePerm)
			if err != nil {
				logrus.Errorf("err1:%s\n", err)
			}
		} else {
			logrus.Errorf("err2:%s\n", err)
		}
	} else {
		if !s.IsDir() {
			// logrus.Errorf("err:%s\n", "The specified path is not a directory.")
		}
	}
	if !strings.HasSuffix(directory, "/") {
		directory = directory + "/"
	}
	filePath := directory + fileName

	return filePath
}

func Md5SumFile(file string) string {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return err.Error()
	}
	md5 := md5.New()
	md5.Write(data)
	md5Data := md5.Sum([]byte(nil))
	return hex.EncodeToString(md5Data)
}
