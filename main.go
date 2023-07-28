package main

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

var DB *sql.DB

// 图片上传服务
// PUT方法上传图片t不同
// 图床服务器
func handleRequest(w http.ResponseWriter, r *http.Request) {
	// 解析表单数据
	err := r.ParseMultipartForm(10 << 20) // 限制上传文件的大小
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	a := r.FormValue("bucket-name")
	fmt.Println(a)

	// 获取上传的文件
	file, handler, err := r.FormFile("file")

	fmt.Println(handler.Filename, "文件名称")

	// 图片数据写入mysql
	all, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println("读取文件失败", err)
		return
	}

	img := Image{
		ObjId: 1,
		Data:  all,
	}

	err = writeBinaryData(img)
	if err != nil {
		fmt.Println("write to mysql error: ", err)
	}

	// 返回成功的响应
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Image uploaded successfully"))
}

type Image struct {
	ID    int
	ObjId int
	Data  []byte
}

func writeBinaryData(image Image) error {
	stmt, err := DB.Prepare("INSERT INTO img (id, obj_id, data) VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(image.ID, image.ObjId, image.Data)
	if err != nil {
		return err
	}

	return nil
}

func createDBConnection() error {
	db, err := sql.Open("mysql", "root:mengjiale@tcp(127.0.0.1:3306)/obs")
	if err != nil {
		return err
	}
	DB = db
	return err
}

func main() {

	// 连接数据库
	err := createDBConnection()
	if err != nil {
		fmt.Println(err)
		return
	}

	http.HandleFunc("/", handleRequest)
	http.HandleFunc("/get", handlerGet)
	// 启动服务
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("服务启动失败")
		return
	}
}

func handlerGet(writer http.ResponseWriter, request *http.Request) {
	img := Image{}
	id := request.URL.Query().Get("id")
	DB.QueryRow("select * from img where id = ?", id).Scan(&img.ID, &img.ObjId, &img.Data)

	// 将文件内容复制到目标文件
	err := os.WriteFile("a.jpg", img.Data, 0644)
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
