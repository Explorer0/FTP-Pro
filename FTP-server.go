package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type User struct {
	login 	net.Conn
	name  	string
	pwd	  	string
	workDir	string
	status  int
}

var funcMap=make(map[string]func(*User,string,string)(uint,string,error))       //func(user,cmd,arg)(code,description,error)
var responseCodeMap=make(map[uint]string)

func main(){
	//初始化FTP的命令映射函数表，响应码映射说明表
	initFTP()

	var listenAddress="0.0.0.0:8000"
	//第二个参数为服务器监听端口号
	if len(os.Args)>=2{
		port,_:=strconv.Atoi(os.Args[1])
		if port<=65535 && port>=0{
			listenAddress="0.0.0.0:"+os.Args[1]
		}
	}

	ftpListener,err:=net.Listen("tcp",listenAddress)
	if err!=nil{
		log.Fatal(err)
	}

	for {
		conn,err:=ftpListener.Accept()
		if err!=nil{
			log.Print(err)
			continue
		}
		//开启线程处理新用户的连接
		go handlerFTPConn(conn)
	}
}

func handlerFTPConn(c net.Conn){
	cmdBuf:=make([]byte,1024)
	var tmpUsr User

	c.Write([]byte("220 "+""+responseCodeMap[220]+"\r\n"))          //发送Welcome消息
	defer c.Close()

	for{
		//接收命令
		n,err:=c.Read(cmdBuf)
		if err!=nil{
			log.Print(err)
			return
		}
		cmd:=strings.TrimSpace(strings.Split(string(cmdBuf[:n])," ")[0])
		arg:=""
		if len(strings.Split(string(cmdBuf[:n])," "))>1{
			arg=strings.TrimSpace(strings.Split(string(cmdBuf[:n])," ")[1])
		}

		//根据cmd寻找处理函数
		//1.无此cmd的处理函数
		//2.还未登录状态
		processFunc,ok:=funcMap[cmd]
		if !ok{
			processFunc=defaultProcessFunc
		}else if tmpUsr.status<1 && (cmd=="USER" || cmd=="PASS"){
			processFunc=loginFunc
		}

		code,arg,err:=processFunc(&tmpUsr,cmd,arg)
		c.Write([]byte(fmt.Sprintf("%d ",code)+responseCodeMap[code]+" "+arg+"\r\n"))
		//QUIT的response code
		if code==426{
			break
		}
	}
}
func initFTP(){
	responseCodeMap[220]="(Fucker FTP 0.0.0)"
	responseCodeMap[404]="Unknown cmd!"
	responseCodeMap[426]="Close the connection."
	responseCodeMap[331]="Please specify the password."
	responseCodeMap[530]="Please login with USER and PASS."
	responseCodeMap[230]="Login successful."
	responseCodeMap[250]="Directory successfully changed."
	responseCodeMap[200]="Switching to ASCII mode."


	funcMap["EXIT"]=exitFunc
	funcMap["QUIT"]=exitFunc
	funcMap["USER"]=loginFunc
	funcMap["PASS"]=loginFunc
	funcMap["CWD"] =changDirFunc

}
func defaultProcessFunc(curUsr *User, cmd, arg string)(uint,string,error){
	return 404,"("+cmd+")",nil
}
func exitFunc(curUsr *User, cmd, arg string)(uint,string,error){
	return 426,"",nil
}
func loginFunc(curUsr *User, cmd, arg string)(uint,string,error){
	if cmd=="USER"{
		curUsr.name=arg
		return 331,"",nil
	}else if  cmd=="PASS"{
		curUsr.pwd=arg
		if checkUser(curUsr.name,curUsr.pwd) {
			curUsr.status=1
			return 230, "", nil
		}
		return 530, "(login incorrect)",nil
	}else{
		return 530,"",nil
	}
}
func changDirFunc(curUsr *User, cmd, arg string)(uint,string,error){
	cmd+=" "+arg
	_,cmdErr:=exec.Command(cmd).Output()
	if cmdErr!=nil{
		return 451,"(Unknown directory:"+arg+")",cmdErr
	}
	curUsr.workDir,_=getCurrentWorkPath()
	curUsr.workDir+="/"
	return 250,"",nil
}
func getCwdFunc(curUsr *User, cmd, arg string)(uint,string,error){
	cwd,err:=getCurrentWorkPath()
	if err!=nil{
		return 451,"Can't get current work directory.",err
	}

}
func getCurrentWorkPath()(string,error){
	dirPaht,err:=os.Getwd()
	if err!=nil{
		return "",err
	}
	return dirPaht,nil
}
func checkUser(name, pwd string)bool{
	return true
}