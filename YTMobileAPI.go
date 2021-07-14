package main

import "C"
import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/electricbubble/guia2"
	ecrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/mr-tron/base58"
	"github.com/sirupsen/logrus"
	"github.com/yottachain/YTCoreService/api"
	"github.com/yottachain/YTCoreService/env"
	"github.com/yottachain/YTMobileAPI/aes"
	"github.com/yottachain/YTMobileAPI/router"
	"github.com/yottachain/YTMobileAPI/tools"
	"golang.org/x/crypto/ripemd160"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unsafe"
)

//export Register
func Register(userName, privateKey string) *C.char {
	os.Setenv("YTFS.snlist", "conf/snlist.properties")
	env.SetVersionID("2.0.1.5")
	api.StartMobileAPI()
	_, err := api.NewClientV2(&env.UserInfo{
		UserName: userName,
		Privkey:  []string{privateKey}}, 3)
	if err != nil {
		logrus.Panicf(":%s\n", err)

	}

	return C.CString("用户: " + userName + " 注册成功")
}

//export GetPubKey
func GetPubKey(userName string) *C.char {
	privateKey, publicKey := createKey()
	user := tools.User{
		UserName:   userName,
		PrivateKey: privateKey,
		PublicKey:  "YTA" + publicKey,
	}
	var data []byte
	newData := C.GoBytes(unsafe.Pointer(&data), 0)
	newData, _ = json.Marshal(user)
	//defer C.free(unsafe.Pointer(newData))
	//if err!=nil{
	//	logrus.Errorf("Marshal err:%s\n", err)
	//}
	UserWrite(newData)
	//logrus.Infof(user.PublicKey)
	return C.CString(user.PublicKey)
}

//export UploadObject
func UploadObject(url, filePath, bucketName, userName, privateKey string) {
	//c := api.GetClientByName(userName)
	//var filename string
	//filename = filepath.Base(filePath)
	//
	//do:= c.UploadPreEncode(bucketName,filename)
	//err:=do.UploadFile(filePath)
	//if err!=nil {
	//	logrus.Errorf(":%s\n", err)
	//	return
	//}
	env.SetVersionID("2.0.1.5")
	var fileName string
	fileName = filepath.Base(filePath)
	os.Setenv("YTFS.snlist", "conf/snlist.properties")
	api.StartMobileAPI()
	c, err := api.NewClientV2(&env.UserInfo{
		UserName: userName,
		Privkey:  []string{privateKey}}, 3)
	if err != nil {
		logrus.Panicf(":%s\n", err)
	}
	do := c.UploadPreEncode(bucketName, fileName)

	err1 := do.UploadFile(filePath)

	if err1 != nil {
		logrus.Panicf("err1: %s", err1)
	}
	ss := do.OutPath()
	//up,err1:= api.NewUploadEncObject(ss)
	//if err1 != nil {
	//	logrus.Errorf(":%s\n", err1)
	//}
	//err1 = up.Upload()
	//if err1 != nil {
	//	logrus.Errorf(":%s\n", err1)
	//}
	newUrl := url + "/api/v1/saveFileToLocal"
	postFile(ss, newUrl)
}

func postFile(filename string, targetUrl string) error {
	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)

	//关键的一步操作
	fileWriter, err := bodyWriter.CreateFormFile("uploadfile", filename)
	if err != nil {
		fmt.Println("error writing to buffer")
		return err
	}

	//打开文件句柄操作
	fh, err := os.Open(filename)
	if err != nil {
		fmt.Println("error opening file")
		return err
	}
	defer fh.Close()

	//iocopy
	_, err = io.Copy(fileWriter, fh)
	if err != nil {
		return err
	}

	contentType := bodyWriter.FormDataContentType()
	bodyWriter.Close()

	resp, err := http.Post(targetUrl, contentType, bodyBuf)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	resp_body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	fmt.Println(resp.Status)
	fmt.Println(string(resp_body))
	return nil
}

//export AddUserToS3Server
func AddUserToS3Server(url, username, publicKey string) uint32 {
	var num uint32

	newUrl := url + "/api/v1/addPubkey?publicKey=" + publicKey + "&userName=" + username
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	//http cookie接口
	cookieJar, _ := cookiejar.New(nil)
	c := &http.Client{
		Jar:       cookieJar,
		Transport: tr,
	}

	resp, err := c.Get(newUrl)
	if err != nil {

	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err == nil {
		err = json.Unmarshal(body, &num)
	} else {

	}
	return num
}

//export DownloadObject
func DownloadObject(url, filePath, fileName, bucketName string) *C.char {

	userdata := tools.ReadUserInfo()
	var user tools.User
	user = tools.UserUnmarshal(userdata)

	userName := user.UserName
	var blockNum int
	blockNum = 0

	key, err := aes.NewKey(user.PrivateKey, user.Num)
	if err != nil {

	}
	index := strings.Index(fileName, "/")
	if index != -1 {
		directory := filePath + "/" + bucketName + "/" + fileName[:index]
		fname := fileName[index+1:]
		filePath = tools.CreateDirectory(directory, fname)
	} else {
		directory := filePath + "/" + bucketName
		filePath = tools.CreateDirectory(directory, fileName)
	}

	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		logrus.Errorf("erra:%s\n", err)
	}
	defer f.Close()

	for blockNum != -1 {
		data, err := DownBlock(url, userName, bucketName, fileName, blockNum)
		if err != nil {
			blockNum = -1
			//g.JSON(http.StatusAccepted, gin.H{"Msg": "[" + fileName + "] download is failure ."})
			return C.CString("[" + fileName + "] download is failure .")
		} else {
			if len(data) > 0 {

				block := aes.NewEncryptedBlock(data)
				err1 := block.Decode(key, f)

				if err1 != nil {
					fmt.Println(err1)
					//g.JSON(http.StatusAccepted, gin.H{"Msg": "[" + fileName + "] download is failure ."})
					return C.CString("[" + fileName + "] download is failure .")
				} else {
					blockNum++
				}

			} else {
				blockNum = -1
			}
		}
	}
	md5Value := tools.Md5SumFile(filePath)

	return C.CString("File md5:" + md5Value + " ,[" + fileName + " ] download is successful.")
}

//func UploadObject(filePath)

func UserWrite(data []byte) {
	fp, err := os.OpenFile("user.json", os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		logrus.Errorf("err:%s\n", err)
	}
	defer fp.Close()
	_, err = fp.Write(data)
	if err != nil {
		logrus.Errorf("err:%s\n", err)
	}
}

func DownBlock(url, userName, bucketName, fileName string, blockNum int) ([]byte, error) {

	str2 := fmt.Sprintf("%d", blockNum)
	newUrl := url + "/api/v1/getBlockForSGX?userName=" + userName + "&bucketName=" + bucketName + "&fileName=" + fileName + "&blockNum=" + str2

	var data []byte

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	//http cookie接口
	cookieJar, _ := cookiejar.New(nil)
	c := &http.Client{
		Jar:       cookieJar,
		Transport: tr,
	}
	resp, err := c.Get(newUrl)

	if err != nil {
		return nil, err
	} else {
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println(err)
		} else {
			err = json.Unmarshal(body, &data)
			//data = body
		}

	}
	return data, nil
}

var (
	privKey string
	pubKey  string
)

func createKey() (string, string) {
	privKey, _ := ecrypto.GenerateKey()
	privKeyBytes := ecrypto.FromECDSA(privKey)
	rawPrivKeyBytes := append([]byte{}, 0x80)
	rawPrivKeyBytes = append(rawPrivKeyBytes, privKeyBytes...)
	checksum := sha256Sum(rawPrivKeyBytes)
	checksum = sha256Sum(checksum)
	rawPrivKeyBytes = append(rawPrivKeyBytes, checksum[0:4]...)
	privateKey := base58.Encode(rawPrivKeyBytes)

	pubKey := privKey.PublicKey
	pubKeyBytes := ecrypto.CompressPubkey(&pubKey)
	checksum = ripemd160Sum(pubKeyBytes)
	rawPublicKeyBytes := append(pubKeyBytes, checksum[0:4]...)
	publicKey := base58.Encode(rawPublicKeyBytes)
	return privateKey, publicKey
}
func sha256Sum(bytes []byte) []byte {
	h := sha256.New()
	h.Write(bytes)
	return h.Sum(nil)
}
func ripemd160Sum(bytes []byte) []byte {
	h := ripemd160.New()
	h.Write(bytes)
	return h.Sum(nil)
}

func checkErr(err error, msg ...string) {
	if err == nil {
		return
	}

	var output string
	if len(msg) != 0 {
		output = msg[0] + " "
	}
	output += err.Error()
	log.Fatalln(output)
}

func waitForElement(driver *guia2.Driver, bySelector guia2.BySelector) (element *guia2.Element, err error) {
	var ce error
	exists := func(d *guia2.Driver) (bool, error) {
		element, ce = d.FindElement(bySelector)
		if ce == nil {
			return true, nil
		}
		// 如果直接返回 error 将直接终止 `driver.Wait`
		return false, nil
	}
	if err = driver.Wait(exists); err != nil {
		return nil, fmt.Errorf("%s: %w", err.Error(), ce)
	}
	return
}

func main() {
	//version, err := syscall.GetVersion()
	//if err != nil {
	//
	//}
	//fmt.Printf("%d.%d(%d)", byte(version), uint8(version>>8), version>>16)

	//GetPubKey("yottanewsabc")
	//Register("yottanewsabc","5HyFZaX8TecHEp5wigibc8yPbadypGUCGWjBf5Yo3xtmN4mPJnn")
	//UploadObject("http://117.161.72.89:8080", "E:\\安装包\\node-v14.15.1-x64.msi", "test", "yottanewsabc", "5HyFZaX8TecHEp5wigibc8yPbadypGUCGWjBf5Yo3xtmN4mPJnn")

	//err := http.ListenAndServe("0.0.0.0:8989",nil)
	//if err!= nil {
	//	logrus.Panicf("http listen failed")
	//}
	logrus.Infof(time.Now().Format("2006-01-02 15:04:05") + "strart ......")
	flag.Parse()
	router := router.InitRouter()

	err1 := router.Run(":5551")
	if err1 != nil {
		panic(err1)
	}
	logrus.Info("strart ......")
}
