package httpServer

import "C"
import (
	"crypto/sha256"
	"crypto/tls"
	"encoding/json"
	"fmt"
	ecrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/gin-gonic/gin"
	"github.com/mr-tron/base58"
	"github.com/sirupsen/logrus"
	"github.com/yottachain/YTMobileAPI/aes"
	"github.com/yottachain/YTMobileAPI/tools"
	"golang.org/x/crypto/ripemd160"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strings"
)

func AddUserToS3server(g *gin.Context) {
	userName := g.Query("userName")
	publicKey := g.Query("publicKey")
	url := g.Query("url")
	var num uint32
	num, err := AddKey(userName, publicKey, url)

	if err != nil {
		g.JSON(http.StatusBadRequest, gin.H{"errMsg": err})
	} else {
		//user := tools.User{
		//	UserName:   uu.UserName,
		//	Num:        num,
		//	PrivateKey: uu.PrivateKey,
		//	PublicKey:  uu.PublicKey,
		//}
		//
		//data, err := json.Marshal(user)
		//if err != nil {
		//	logrus.Errorf("Marshal err:%s\n", err)
		//}
		//
		//tools.UserWrite(data)

		g.JSON(http.StatusOK, gin.H{"publicKey:": publicKey, "userName": userName, "userNum": num})
	}
}

func AddKey(username, pubKey, url string) (uint32, error) {
	var num uint32

	newurl := url + "/api/v1/addPubkey?publicKey=" + pubKey + "&userName=" + username

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	//http cookie接口
	cookieJar, _ := cookiejar.New(nil)
	c := &http.Client{
		Jar:       cookieJar,
		Transport: tr,
	}

	resp, err := c.Get(newurl)
	if err != nil {

	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err == nil {
		err = json.Unmarshal(body, &num)
	} else {

	}

	return num, err
}

func GetPubKey(g *gin.Context) {
	userName := g.Query("userName")

	privateKey, publicKey := createKey()
	user := tools.User{
		UserName:   userName,
		PrivateKey: privateKey,
		PublicKey:  "YTA" + publicKey,
	}
	g.JSON(http.StatusOK, gin.H{"publicKey": user.PublicKey, "privateKey": user.PrivateKey, "userName": user.UserName})
}

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
func DownloadObject(g *gin.Context) {
	url := g.Query("url")
	filePath := g.Query("filePath")
	fileName := g.Query("fileName")
	bucketName := g.Query("bucketName")
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
			//return C.CString("[" + fileName + "] download is failure .")
		} else {
			if len(data) > 0 {

				block := aes.NewEncryptedBlock(data)
				err1 := block.Decode(key, f)

				if err1 != nil {
					fmt.Println(err1)
					//g.JSON(http.StatusAccepted, gin.H{"Msg": "[" + fileName + "] download is failure ."})
					//return C.CString("[" + fileName + "] download is failure .")
				} else {
					blockNum++
				}

			} else {
				blockNum = -1
			}
		}
	}
	md5Value := tools.Md5SumFile(filePath)

	//return C.CString("File md5:" + md5Value + " ,[" + fileName + " ] download is successful.")
	g.JSON(http.StatusOK, gin.H{"File md5:": md5Value, "Msg": "[" + fileName + "] download is successful.", "filePath": filePath})
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
