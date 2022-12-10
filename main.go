package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/oklog/ulid/v2"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/go-sql-driver/mysql"

	"github.com/joho/godotenv"
	_ "github.com/oklog/ulid/v2" // I might not use this
)

type UserResForHTTPGet struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

type GoStruct struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

type Ping struct {
	Id string `json:"id"`
}

// ① GoプログラムからMySQLへ接続
var db *sql.DB

func init() {
	// ①-1

	// ここで.envファイル全体を読み込みます。
	// この読み込み処理がないと、個々の環境変数が取得出来ません。
	// 読み込めなかったら err にエラーが入ります。
	err := godotenv.Load(".env")

	// もし err がnilではないなら、"読み込み出来ませんでした"が出力されます。
	if err != nil {
		fmt.Printf("読み込み出来ませんでした: %v", err)
	}

	// .envの SAMPLE_MESSAGEを取得して、messageに代入します。
	mysqlUser := os.Getenv("mysqlUser")
	mysqlUserPwd := os.Getenv("mysqlUserPwd")
	mysqlDatabase := os.Getenv("mysqlDatabase")

	log.Println(mysqlUser)
	log.Println(mysqlUserPwd)
	log.Println(mysqlDatabase)

	// ①-2
	_db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@(localhost:3306)/%s", mysqlUser, mysqlUserPwd, mysqlDatabase))
	if err != nil {
		log.Fatalf("fail: sql.Open, %v\n", err)
	}
	// ①-3
	if err := _db.Ping(); err != nil {
		log.Fatalf("fail: _db.Ping, %v\n", err)
	}
	db = _db
}

// ② /userでリクエストされたらnameパラメーターと一致する名前を持つレコードをJSON形式で返す
func handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	switch r.Method {
	case http.MethodOptions:
		w.WriteHeader(200)
		return
	case http.MethodGet:
		/* フロントエンドとバックエンドの接続では全データを参照するので、絞り込みは必要はなさそう
		// ②-1
		name := r.URL.Query().Get("name") // To be filled
		if name == "" {
			log.Println("fail: name is empty")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		*/

		// ②-2
		rows, err := db.Query("SELECT id, name, age FROM user ")
		if err != nil {
			log.Printf("fail: db.Query, %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		//10/19/1902 i do not know how i can load the request body cuz right now there is the sign of 404
		// ②-3
		users := make([]UserResForHTTPGet, 0)
		for rows.Next() {
			var u UserResForHTTPGet
			if err := rows.Scan(&u.Id, &u.Name, &u.Age); err != nil {
				log.Printf("fail: rows.Scan, %v\n", err)

				if err := rows.Close(); err != nil { // 500を返して終了するが、その前にrowsのClose処理が必要
					log.Printf("fail: rows.Close(), %v\n", err)
				}
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			users = append(users, u)
		}

		// ②-4
		bytes, err := json.Marshal(users)
		if err != nil {
			log.Printf("fail: json.Marshal, %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Write(bytes)
	default:
		log.Printf("fail: HTTP Method is %s\n", r.Method) // ポイント１/6
		w.WriteHeader(http.StatusBadRequest)
		return

	case http.MethodPost:

		// ②-1 getting name . if name do not exist "name is empty" is output
		//name := r.FormValue("name") // To be filled
		//age := r.FormValue("age")
		_body := r.Body

		var stcData GoStruct

		buf := new(bytes.Buffer)
		buf.ReadFrom(_body)
		body := buf.String() //io.reader をstringに変換

		if err := json.Unmarshal([]byte(body), &stcData); err != nil {
			fmt.Println(err)

		}
		fmt.Printf("%+v\n", stcData) //go kouzoutai に変化する

		_id := ulid.Make() //TODO:stringに直す必要がある
		id := _id.String()

		name := stcData.Name
		age := stcData.Age

		if name == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if len(name) > 50 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if age < 20 || age > 80 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		//age := r.FormValue("age")
		/*
			if name == "" {
				log.Println("fail: name is empty")
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if age == "" {
				log.Println("fail: age is empty")
				w.WriteHeader(http.StatusBadRequest)
				return
			}
		*/

		// 10/20　9時50分前に拾ってきたサイト　todo  insert
		ins, err := db.Prepare("INSERT INTO user(id,name,age) VALUES(?,?,?)")
		if err != nil {
			log.Fatal(err)
		}
		ins.Exec(id, name, age) // first test with a string  ←maybe I need to struct a
		defer ins.Close()       //　なんかわからんけど、すごい必要だった！！
		//rows, err := db.Query("INSERT INTO user (id,name,age) VALUES (%s,%s,%d)", id, name, age)

		if err != nil {
			log.Printf("fail: db.Query, %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err == nil {
			w.WriteHeader(http.StatusOK)
			log.Println(id)
			log.Println(name)
			log.Println(age)

			ping := Ping{id}
			bytess, err := json.Marshal(ping)

			if err != nil {
				log.Printf("fail: json.Marshal, %v\n", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			w.Write(bytess) //todo

			return

		}

		// ②-3 ここ要らなくね？
		/*
			users := make([]UserResForHTTPGet, 0)
			for rows.Next() {
				var u UserResForHTTPGet
				if err := rows.Scan(&u.Id, &u.Name, &u.Age); err != nil {
					log.Printf("fail: rows.Scan, %v\n", err)

					if err := rows.Close(); err != nil { // 500を返して終了するが、その前にrowsのClose処理が必要
						log.Printf("fail: rows.Close(), %v\n", err)
					}
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				users = append(users, u)
			}



		*/

		// ②-4 might be necessary

		//=todo idを返しているか確認

	}
}

func main() {
	// ② /userでリクエストされたらnameパラメーターと一致する名前を持つレコードをJSON形式で返す
	http.HandleFunc("/user", handler)

	// ③ Ctrl+CでHTTPサーバー停止時にDBをクローズする
	closeDBWithSysCall()

	// 8000番ポートでリクエストを待ち受ける
	log.Println("Listening...")
	if err := http.ListenAndServe(":8000", nil); err != nil {
		log.Fatal(err)

	}
}

// ③ Ctrl+CでHTTPサーバー停止時にDBをクローズする
func closeDBWithSysCall() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		s := <-sig
		log.Printf("received syscall, %v", s)

		if err := db.Close(); err != nil {
			log.Fatal(err)
		}
		log.Printf("success: db.Close()")
		os.Exit(0)
	}()
}
