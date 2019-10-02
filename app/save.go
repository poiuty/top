package main

import (
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	
	"strconv"
)

type tableRoom struct {
	Id     int64   `db:"id"`
	Name   string  `db:"name"`
	Gender int     `db:"gender"`
	Last   int64   `db:"last"`
}

type tableDonator struct {
	Id    int64    `db:"id"`
	Name  string  `db:"name"`
}

type saveData struct {
	room, donator string
	token, online int64
}

type Save struct {
	donate chan *saveData
	online chan *saveData
}

func saveOnline(conn *sqlx.DB, rid, online int64){
	conn.Exec("INSERT INTO `online` (`rid`, `online`, `time`) VALUES (?, ?, unix_timestamp(now()))", rid, online)
}

func saveDonate(conn *sqlx.DB, did, rid, token int64) {
	conn.Exec("INSERT INTO `stat` (`did`, `rid`, `token`, `time`) VALUES (?, ?, ?, unix_timestamp(now()))", did, rid, token)
}

func getDonatorID(conn *sqlx.DB, name string) int64 {
	var donator tableDonator
	err := conn.Get(&donator, "SELECT * FROM donator WHERE name=?", name)
	if err != nil {		
		res, _ := conn.Exec("INSERT INTO donator (`name`) VALUES (?)", name)
		id, _ := res.LastInsertId()
		return id
	}
	return donator.Id
}

func updateWorker(conn *sqlx.DB, id int64) bool {
	_, err := conn.Exec("UPDATE room SET last = unix_timestamp(now()) WHERE id =?", id)
	if err != nil {
		return false
	}
	return true
}

func getRoomInfo(conn *sqlx.DB, name string) (tableRoom, bool) {
	result := true
	var room tableRoom
	err := conn.Get(&room, "SELECT * FROM room WHERE name=?", name)
	if err != nil {
		result = false
	}
	return room, result
}

func saveBase(s *Save, h *Hub){
	conn, err := sqlx.Connect("mysql", "user:passwd@unix(/var/run/mysqld/mysqld.sock)/db")
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	for {
		select {
			case info := <-s.donate:
			room, ok := getRoomInfo(conn, info.room)
			if ok {
				updateWorker(conn, room.Id)
				saveDonate(conn, getDonatorID(conn, info.donator), room.Id, info.token);
				if info.token >= 100 {
					token := strconv.FormatInt(info.token, 10)
					h.broadcast <- []byte(info.donator+" send "+token+" tokens to "+info.room)
				}
			}
			case info := <-s.online:
			room, ok := getRoomInfo(conn, info.room)
			if ok {
				saveOnline(conn, room.Id, info.online);
			}
		}
	}
}

func sendPost(room, name, token, online string) {
	t, _ :=  strconv.ParseInt(token, 10, 64)
	o, _ :=  strconv.ParseInt(online, 10, 64)
	
	data := &saveData{room: room, donator: name, token: t, online: o}
	saveStat.donate <- data
}
