package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

const (
	BUCKET_ID  = "bucket-id"
	FORM_FIELD = "file"
)

var DB *sql.DB

type Image struct {
	Id       int
	BucketId int
	Name     string
	Md5      string
	Data     []byte
}

type Response struct {
	Url string
}

// 图片上传服务
// PUT方法上传图片t不同
// 图床服务器

func SaveFile(w http.ResponseWriter, r *http.Request) {
	// 解析表单数据
	err := r.ParseMultipartForm(10 << 20) // 限制上传文件的大小
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	bucketId := getBucketId(w, r, err)
	if bucketId == 0 {
		fmt.Println("bucket-id is empty please check.")
		return
	}

	// 获取上传的文件
	file, handler, err := r.FormFile(FORM_FIELD)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	name := handler.Filename

	// 图片数据写入mysql
	all, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println("读取文件失败", err)
		return
	}

	md5 := CalculateMD5(all)
	img := Image{
		BucketId: bucketId,
		Md5:      md5,
		Name:     name,
		Data:     all,
	}

	err = writeBinaryData(img)
	if err != nil {
		fmt.Println("write to mysql error: ", err)
	}

	response := Response{
		Url: fmt.Sprintf("http://127.0.0.1:8080/ofs/%s/%s.jpg", r.FormValue(BUCKET_ID), md5),
	}
	resp, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(resp)
	if err != nil {
		return
	}
}

func getBucketId(w http.ResponseWriter, r *http.Request, err error) int {
	// string convert to int
	bucketId, err := strconv.Atoi(r.FormValue(BUCKET_ID))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return 0
	}
	return bucketId
}

func writeBinaryData(image Image) error {
	stmt, err := DB.Prepare("INSERT INTO img (bucket_id, name, data, md5) VALUES (?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(image.BucketId, image.Name, image.Data, image.Md5)
	if err != nil {
		return err
	}

	return nil
}

func createConnection() error {
	config := GetDBConfig()
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", config.UserName, config.Password,
		config.Host, config.Port, config.Db))
	if err != nil {
		return err
	}
	DB = db
	return err
}

func GetPicByUrl(writer http.ResponseWriter, request *http.Request) {
	img := Image{}
	path := request.URL.Path
	bucketId, _ := strconv.Atoi(strings.Split(path, "/")[2])
	md5 := strings.Split(path, "/")[3]
	fmt.Println(bucketId, strings.Split(md5, ".")[0])
	err := DB.QueryRow(`select data from img where bucket_id = ? and md5 = ?`, bucketId, strings.Split(md5, ".")[0]).Scan(&img.Data)
	if err != nil {
		fmt.Println(err)
		return
	}

	// 将文件内容复制到目标文件
	err = os.WriteFile("a.jpg", img.Data, 0644)
	if err != nil {
		log.Println("Error copying the file")
		log.Println(err)
		return
	}

	// 设置响应头
	writer.Header().Set("Content-Type", "image/jpeg")

	// 写入图片内容作为响应
	n, err := writer.Write(img.Data)
	fmt.Println("写入数据", n)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
}

func main() {
	// 连接数据库
	err := createConnection()
	if err != nil {
		panic(fmt.Sprintf("create db connection error: %v", err))
	}

	http.HandleFunc("/ofs/put", SaveFile)
	http.HandleFunc("/ofs/", GetPicByUrl)

	// 启动服务
	fmt.Println("服务启动......")
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("服务启动失败", err)
		return
	}
}
