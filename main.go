package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

const (
	BucketId  = "bucket-id"
	FormField = "file"
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

// 获取本机的IP
func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

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
	file, handler, err := r.FormFile(FormField)
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
		Url: fmt.Sprintf("http://%s/ofs/%s/%s.jpg", getLocalIP(), r.FormValue(BucketId), md5),
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
	bucketId, err := strconv.Atoi(r.FormValue(BucketId))
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
	err := DB.QueryRow(`select data from img where bucket_id = ? and md5 = ?`, bucketId, strings.Split(md5, ".")[0]).Scan(&img.Data)
	if err != nil {
		fmt.Println(err)
		return
	}

	// 设置响应头
	writer.Header().Set("Content-Type", "image/jpeg")

	// 写入图片内容作为响应
	_, err = writer.Write(img.Data)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
}

// 实现配置文件变化的时候，服务进行重启
func watchConfig(sigChan chan os.Signal) {
	changed := make(chan bool, 1)
	// 获取配置文件的元数据
	fileInfo, err := os.Stat("config.toml")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("xxxxxxxxxxxxxx", fileInfo.ModTime())

	modifyTime := fileInfo.ModTime()

	go func() {
		timer := time.NewTicker(10 * time.Second)

		for {
			f, err := os.Stat("config.toml")
			if err != nil {
				fmt.Println(err)
			}
			m := f.ModTime()
			fmt.Println("aaaaaaaaaaaa", m.Sub(modifyTime).Seconds())
			if m.Sub(modifyTime).Seconds() > 0 {
				fmt.Println("文件被修改了。。。。。。。")
				changed <- true
				modifyTime = m
				sigChan <- syscall.SIGHUP
			}
			select {
			case <-changed:
				fmt.Println("配置文件发生的变化")

			case <-timer.C:
				fmt.Println("定时器触发")
			}
		}

	}()
}

func main() {
	// 连接数据库
	err := createConnection()
	if err != nil {
		panic(fmt.Sprintf("create db connection error: %v", err))
	}

	http.HandleFunc("/ofs/put", SaveFile)
	http.HandleFunc("/ofs/", GetPicByUrl)

	sigalChan := make(chan os.Signal)
	signal.Notify(sigalChan, syscall.SIGHUP)

	watchConfig(sigalChan)

	// 启动服务
	server := &http.Server{
		Addr:    ":8080",
		Handler: nil,
	}

	for {
		go server.ListenAndServe()
		<-sigalChan

		fmt.Println("signal received, shutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		err = server.Shutdown(ctx)

		if err != nil {
			fmt.Println("服务关闭失败", err)
			return
		} else {
			fmt.Println("服务开始重启。。。。。。。")
		}
	}

}
