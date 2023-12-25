package main

import (
	"bytes"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

var (
	port                     string
	contractAddress          string
	abiStr                   string
	contractMethodId         string
	executeRequestTaskStatus bool
	dbase                    string
	dbTablePrefix            string
	dbUsername               string
	dbPassword               string
	dbPort                   string
	dbHost                   string
	rpcUrl                   string
	accountTaskStatus        string
	pullTaskStatus           string
)

type Input struct {
	Address string `json:"address"`
}

type Response struct {
	Data    string `json:"data"`
	Message string `json:"msg"`
	Code    int    `json:"code"`
}

type ResponseList struct {
	Data interface{} `json:"data"`
	Msg  string      `json:"msg"`
	Code int         `json:"code"`
}

type QueryList struct {
	List     interface{} `json:"params"`
	Page     int         `json:"page"`
	PageSize int         `json:"pagesize"`
	Total    int         `json:"total"`
}

type AccountQueryList struct {
	List     []AccountResponse `json:"list"`
	Page     int               `json:"page"`
	PageSize int               `json:"pagesize"`
	Total    int               `json:"total"`
}

type JSONRPCRequest struct {
	Jsonrpc string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	Id      int           `json:"id"`
}

type JSONRPCResponse struct {
	Result json.RawMessage `json:"result"`
	Error  interface{}     `json:"error"`
	Id     int             `json:"id"`
}

type Account struct {
	Address string
}

type BlockInfo struct {
	Hash         string        `json:"hash"`
	Number       int           `json:"number"`
	Transactions []Transaction `json:"transactions"`
}

type Transaction struct {
	BlockLimit int    `json:"blockLimit"`
	ChainID    string `json:"chainID"`
	ExtraData  string `json:"extraData"`
	From       string `json:"from"`
	GroupID    string `json:"groupID"`
	Hash       string `json:"hash"`
	ImportTime int    `json:"importTime"`
	Input      string `json:"input"`
	To         string `json:"to"`
}

type TransactionReceipt struct {
	Hash    string `json:"transactionHash"`
	From    string `json:"from"`
	To      string `json:"to"`
	Status  int    `json:"status"`
	Input   string `json:"input"`
	Output  string `json:"output"`
	GasUsed string `json:"gasUsed"`
}

type TransactionResponse struct {
	BlockNumber string `json:"block_num"`
	Hash        string `json:"trans_hash"`
	From        string `json:"from"`
	To          string `json:"to"`
	Status      int    `json:"status"`
	// DecodeInput  string `json:"decode_input"`
	// DecodeOutput string `json:"decode_output"`
	Input      string `json:"input"`
	Output     string `json:"output"`
	ImportTime int    `json:"import_time"`
	HeartRate  string `json:"heart_rate"`
	BreathRate string `json:"breath_rate"`
	SleepState int    `json:"sleep_state"`
	PersonId   int    `json:"person_id"`
	// Input       string `json:"input"`
	// Output      string `json:"output"`
	// GasUsed     string `json:"gasUsed"`
	ContactName     string `json:"contact_name"`
	ContactIdentity string `json:"contact_identity"`
}

type TransactionResResponse struct {
	BlockNumber     string `json:"block_num"`
	Hash            string `json:"trans_hash"`
	From            string `json:"from"`
	To              string `json:"to"`
	Status          int    `json:"status"`
	Input           string `json:"input"`
	Output          string `json:"output"`
	ImportTime      int    `json:"import_time"`
	HeartChange     string `json:"heart_change"`
	SleepBreathing  string `json:"sleep_breathing"`
	PersonId        int    `json:"person_id"`
	ContactName     string `json:"contact_name"`
	ContactIdentity string `json:"contact_identity"`
}

type AccountResponse struct {
	Address   string `json:"address"`
	Balance   int    `json:"balance"`
	Cred      int    `json:"cred"`
	ShareNnum int    `json:"share_num"`
}

type SQL struct {
	db *sql.DB
}

func main() {
	initEnvConfig()
	initDB()
	initBlockTask()
	initListen()
}

func initListen() {
	http.HandleFunc("/register", handleRequest)
	http.HandleFunc("/contract-address", getContractAddress)
	http.HandleFunc("/getTransByAddress", getTransByAddress)
	http.HandleFunc("/getResByAddress", getResByAddress)
	http.HandleFunc("/accountRanking", accountRanking)
	fmt.Printf("服务端口: %s\n", port)
	fmt.Printf("当前Cred合约地址: %s\n", contractAddress)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatal("Failed to start server:", err)
	}

}

func initBlockTask() {
	fmt.Printf("加载区块同步任务...\n")
	executeRequestTaskStatus = false
	// 启动异步任务
	if pullTaskStatus == "on" {
		go executeRequestTask()
	}
	if accountTaskStatus == "on" {
		go executeAccountTask()
	}

}

func initEnvConfig() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Failed to load .env file:", err)
	}
	port = os.Getenv("PORT")
	dbase = os.Getenv("DB_DATABASE")
	dbUsername = os.Getenv("DB_USERNAME")
	dbPassword = os.Getenv("DB_PASSWORD")
	dbPort = os.Getenv("DB_PORT")
	dbHost = os.Getenv("DB_HOST")
	dbTablePrefix = os.Getenv("DB_PREFIX")
	rpcUrl = os.Getenv("RPC_URL")
	abiStr = os.Getenv("CONTRACT_ABI")
	contractAddress = os.Getenv("CONTRACT_ADDRESS")
	contractMethodId = os.Getenv("CONTRACT_METHOD_SHARADATE")
	accountTaskStatus = os.Getenv("ACCOUNT_TASK_STATUS")
	pullTaskStatus = os.Getenv("PULL_TASK_STATUS")
	if port == "" {
		port = "5924" // 当未在 .env 文件中指定 PORT 时，默认使用 5000
	}
}

func initDB() {
	sql := NewSQL()
	sql.migrate()
	sql.db.Close()
}

func NewSQL() *SQL {
	// 数据库连接
	s := &SQL{}
	dsn := dbUsername + ":" + dbPassword + "@tcp" + "(" + dbHost + ":" + dbPort + ")/" + dbase
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err.Error())
	}
	s.db = db
	return s
}

func (s *SQL) migrate() {
	// 创建数据库
	_, err := s.db.Exec("CREATE DATABASE IF NOT EXISTS " + dbase)
	if err != nil {
		panic(err.Error())
	}

	// 切换到新创建的数据库
	_, err = s.db.Exec("USE " + dbase)
	if err != nil {
		panic(err.Error())
	}

	// 创建数据表
	tableName_1 := dbTablePrefix + "block_number"
	tableSql_1 := fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s (
		id int(11) unsigned NOT NULL AUTO_INCREMENT,
		block_num int(11) NOT NULL DEFAULT 0,
		block_hash varchar(100) NOT NULL DEFAULT '',
		block_transactions longtext DEFAULT '',
		response_code int(8) NOT NULL DEFAULT 0,
		status tinyint(4) NOT NULL DEFAULT 0,
		PRIMARY KEY (id),
		UNIQUE KEY block_num (block_num) USING BTREE
	  ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`, tableName_1)

	_, err = s.db.Exec(tableSql_1)
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("加载表[%v]成功\n", tableName_1)

	tableName_2 := dbTablePrefix + "block_transactions"
	tableSql_2 := fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s (
		id int(11) unsigned NOT NULL AUTO_INCREMENT,
		block_num int(11) NOT NULL DEFAULT 0,
		trans_hash varchar(100) NOT NULL DEFAULT '',
		`+"`from`"+` varchar(100) NOT NULL,
		`+"`to`"+` varchar(100) NOT NULL,
		input longtext DEFAULT '',
		decode_input longtext DEFAULT '',
		is_contract tinyint(4) NOT NULL DEFAULT 0,
		method_id varchar(20) DEFAULT NULL,
		output longtext DEFAULT '',
		decode_output longtext DEFAULT '',
		status int(10) NOT NULL DEFAULT -1,
		gas_used varchar(100) NOT NULL DEFAULT '0',
		import_time bigint(20) unsigned NOT NULL DEFAULT 0,
		PRIMARY KEY (id),
		UNIQUE KEY trans_hash (trans_hash) USING BTREE,
		KEY `+"`from`"+` (`+"`from`"+`) USING BTREE,
		KEY `+"`to`"+` (`+"`to`"+`) USING BTREE
	  ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`, tableName_2)

	_, err = s.db.Exec(tableSql_2)
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("加载表[%v]成功\n", tableName_2)

	tableName_3 := dbTablePrefix + "block_account"
	tableSql_3 := fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s (
		id int(11) unsigned NOT NULL AUTO_INCREMENT,
		address varchar(100) NOT NULL DEFAULT '',
		balance int(10) NOT NULL DEFAULT 0,
		cred int(10) NOT NULL DEFAULT 0,
		share_num int(10) NOT NULL DEFAULT 0,
		PRIMARY KEY (id),
		UNIQUE KEY address (address) USING BTREE
	  ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`, tableName_3)

	_, err = s.db.Exec(tableSql_3)
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("加载表[%v]成功\n", tableName_3)
	fmt.Println("数据库加载成功！")

}

func executeRequestTask() {
	ticker := time.NewTicker(1 * time.Second)
	executeRequestTaskStatus = true
	if executeRequestTaskStatus {
		fmt.Printf("区块同步任务加载完成!\n")
	}
	sql := NewSQL()

	for {
		<-ticker.C // 等待计时器触发
		// 检查最新区块高度
		currentBlockNumber, maxBlockNum := sql.checkBlock()
		if currentBlockNumber == maxBlockNum {
			continue
		}
		difNum := currentBlockNumber - maxBlockNum
		fmt.Printf("检测到高度差异[%v]，最新区块高度[%v]，本地区块高度[%v]，执行区块同步任务。。\n", difNum, currentBlockNumber, maxBlockNum)
		if difNum > 0 {
			maxBlockNum++
			fmt.Printf("读取区块[%v]\n", maxBlockNum)
			sql.synBlockTask(maxBlockNum)
		}
		defer sql.db.Close()
	}

}

func executeAccountTask() {
	ticker := time.NewTicker(60 * time.Second) // 60s
	sql := NewSQL()
	for {
		<-ticker.C // 等待计时器触发
		sql.synAccountTask()
		defer sql.db.Close()
	}
}

func (s *SQL) synAccountTask() {
	// 查询所有 address
	query := `SELECT address FROM bc_block_account`
	rows, err := s.db.Query(query)
	if err != nil {
		panic(err.Error())
	}
	defer rows.Close()

	var accounts []Account

	// 获取查询结果
	for rows.Next() {
		var account Account
		err = rows.Scan(&account.Address)
		if err != nil {
			panic(err.Error())
		}
		accounts = append(accounts, account)
	}

	// 更新 account
	for _, account := range accounts {
		s.synUpdateAccount(account.Address)
	}
}

func (s *SQL) synUpdateAccount(address string) {
	balance := synAccountBalcance(address)
	cred := synAccountCred(address)
	shareNum := s.synAccountShareNum(address)
	// fmt.Printf("%v||%v||%v\n", balance, cred, shareNum)
	_, err := s.db.Exec("UPDATE bc_block_account SET balance = ?, cred = ?, share_num = ? WHERE address = ?",
		balance, cred, shareNum, address)
	if err != nil {
		log.Fatal(err)
	}
}

func synAccountBalcance(address string) int64 {
	// 加载合约
	abi, err := abi.JSON(strings.NewReader(abiStr))
	if err != nil {
		log.Fatal(err)
	}
	toAddress := common.HexToAddress(address)
	// encodeTo, err := hex.DecodeString(to[2:])
	if err != nil {
		log.Fatal(err)
	}

	// ***************  查积分  *********************

	// 将参数按照 ABI 格式编码为字节数组
	data, err := abi.Pack("balance", toAddress)
	if err != nil {
		panic(err)
	}
	// 将字节数组转换为十六进制字符串
	hexData := hexutil.Encode(data)
	// 构建请求体
	requestData := JSONRPCRequest{
		Jsonrpc: "2.0",
		Method:  "call",
		Params: []interface{}{
			"group0",
			"",
			contractAddress,
			hexData,
		},
		Id: 1,
	}
	requestBody, err := json.Marshal(requestData)
	if err != nil {
		fmt.Println("Failed to marshal request data:", err)
		panic(err)
	}

	// 发送 POST 请求
	response, err := http.Post(rpcUrl, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		fmt.Println("Failed to send request:", err)
		panic(err)
	}
	defer response.Body.Close()

	// 读取响应数据
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Println("Failed to read response body:", err)
		panic(err)
	}

	// 解析响应数据
	var jsonResponse JSONRPCResponse
	err = json.Unmarshal(responseBody, &jsonResponse)
	if err != nil {
		fmt.Println("Failed to unmarshal response body:", err)
		panic(err)
	}

	var balanceReceipt TransactionReceipt
	err = json.Unmarshal(jsonResponse.Result, &balanceReceipt)
	if err != nil {
		fmt.Println("Failed to unmarshal result field:", err)
		panic(err)
	}
	// fmt.Printf("output is: %v\n", )

	balance, err := strconv.ParseInt(balanceReceipt.Output[2:], 16, 64)
	if err != nil {
		panic(err.Error())
	}

	// fmt.Println(balance) // 输出十进制结果
	return balance
}

func synAccountCred(address string) int64 {
	// 加载合约
	abi, err := abi.JSON(strings.NewReader(abiStr))
	if err != nil {
		log.Fatal(err)
	}
	toAddress := common.HexToAddress(address)
	// encodeTo, err := hex.DecodeString(to[2:])
	if err != nil {
		log.Fatal(err)
	}

	// ***************  查积分  *********************

	// 将参数按照 ABI 格式编码为字节数组
	data, err := abi.Pack("cred", toAddress)
	if err != nil {
		panic(err)
	}
	// 将字节数组转换为十六进制字符串
	hexData := hexutil.Encode(data)
	// 构建请求体
	requestData := JSONRPCRequest{
		Jsonrpc: "2.0",
		Method:  "call",
		Params: []interface{}{
			"group0",
			"",
			contractAddress,
			hexData,
		},
		Id: 1,
	}
	requestBody, err := json.Marshal(requestData)
	if err != nil {
		fmt.Println("Failed to marshal request data:", err)
		panic(err)
	}

	// 发送 POST 请求
	response, err := http.Post(rpcUrl, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		fmt.Println("Failed to send request:", err)
		panic(err)
	}
	defer response.Body.Close()

	// 读取响应数据
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Println("Failed to read response body:", err)
		panic(err)
	}

	// 解析响应数据
	var jsonResponse JSONRPCResponse
	err = json.Unmarshal(responseBody, &jsonResponse)
	if err != nil {
		fmt.Println("Failed to unmarshal response body:", err)
		panic(err)
	}

	var credReceipt TransactionReceipt
	err = json.Unmarshal(jsonResponse.Result, &credReceipt)
	if err != nil {
		fmt.Println("Failed to unmarshal result field:", err)
		panic(err)
	}

	cred, err := strconv.ParseInt(credReceipt.Output[2:], 16, 64)
	if err != nil {
		panic(err.Error())
	}

	// fmt.Println(cred) // 输出十进制结果
	return cred
}

func (s *SQL) synAccountShareNum(address string) int {
	// 获取总数
	countQuery := "SELECT COUNT(*) FROM bc_block_transactions WHERE method_id = ? AND `status` = 0 AND `from` = ?"
	var total int
	err := s.db.QueryRow(countQuery, contractMethodId, address).Scan(&total)
	if err != nil {
		panic(err)
	}
	return total
}

func (s *SQL) checkBlock() (_currentBlockNumber int, _maxBlockNum int) {
	// 获取最新区块高度
	// 构建请求体
	requestData := JSONRPCRequest{
		Jsonrpc: "2.0",
		Method:  "getBlockNumber",
		Params: []interface{}{
			"group0",
			"",
		},
		Id: 1,
	}
	requestBody, err := json.Marshal(requestData)
	if err != nil {
		fmt.Println("Failed to marshal request data:", err)
		return
	}

	// 发送 POST 请求
	response, err := http.Post(rpcUrl, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		fmt.Println("Failed to send request:", err)
		return
	}
	defer response.Body.Close()

	// 读取响应数据
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Println("Failed to read response body:", err)
		return
	}

	// 解析响应数据
	var jsonResponse JSONRPCResponse
	err = json.Unmarshal(responseBody, &jsonResponse)
	if err != nil {
		fmt.Println("Failed to unmarshal response body:", err)
		return
	}
	currentBlockNumberString := string(jsonResponse.Result)
	num, err := strconv.Atoi(currentBlockNumberString)
	if err != nil {
		fmt.Println("转换失败:", err)
		return
	}
	currentBlockNumber := num
	// fmt.Println("block number", currentBlockNumber)
	// 获取数据库最新高度
	// 查询最大的 block_num 值
	var maxBlockNum int
	err = s.db.QueryRow("SELECT COALESCE(MAX(block_num), 0) FROM bc_block_number").Scan(&maxBlockNum)
	if err != nil {
		panic(err.Error())
	}
	// fmt.Println("最大的 block_num 值为:", maxBlockNum)
	return currentBlockNumber, maxBlockNum
}

func (s *SQL) synBlockTask(block_num int) {
	// 加载合约
	abi, err := abi.JSON(strings.NewReader(abiStr))
	if err != nil {
		fmt.Println("加载合约失败")
		log.Fatal(err)
	}

	// 构建请求体
	requestData := JSONRPCRequest{
		Jsonrpc: "2.0",
		Method:  "getBlockByNumber",
		Params: []interface{}{
			"group0",
			"",
			block_num,
			false,
			false,
		},
		Id: 1,
	}
	requestBody, err := json.Marshal(requestData)
	if err != nil {
		fmt.Println("构建请求体失败", err)
		return
	}

	// 发送 POST 请求
	response, err := http.Post(rpcUrl, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		fmt.Println("Failed to send request:", err)
		return
	}
	defer response.Body.Close()

	// 读取响应数据
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Println("Failed to read response body:", err)
		return
	}

	// 解析响应数据
	var jsonResponse JSONRPCResponse
	err = json.Unmarshal(responseBody, &jsonResponse)
	if err != nil {
		fmt.Println("Failed to unmarshal response body:", err)
		return
	}
	// fmt.Println("Response status code:", response.StatusCode)
	// 解析 result 字段，获取 hash、number 和 transactions
	var blockInfo BlockInfo
	err = json.Unmarshal(jsonResponse.Result, &blockInfo)
	if err != nil {
		fmt.Println("Failed to unmarshal result field:", err)
		return
	}

	// 输出解析结果
	fmt.Println("Hash:", blockInfo.Hash)
	fmt.Println("Number:", blockInfo.Number)
	fmt.Println("Transactions:")
	for _, tx := range blockInfo.Transactions {
		fmt.Printf(" - Hash: %s\n", tx.Hash)
		fmt.Printf("   From: %s\n", tx.From)
		fmt.Printf("   To: %s\n", tx.To)
		// fmt.Printf("   Input: %s\n", tx.Input)
		decode_input := ""
		is_contract := 0
		method_id := ""
		if tx.To == contractAddress {
			// 获取合约方法id
			is_contract = 1
			method_id = tx.Input[2:10]
			decodedSig, err := hex.DecodeString(method_id)
			if err != nil {
				log.Fatal(err)
			}
			if method_id == contractMethodId {
				// 加载合约方法
				method, err := abi.MethodById(decodedSig)
				if err != nil {
					log.Fatal(err)
				}
				decodedData, err := hex.DecodeString(tx.Input[10:])
				if err != nil {
					log.Fatal(err)
				}
				re, err := method.Inputs.Unpack(decodedData)
				if err != nil {
					// fmt.Printf("解码失败：%v\n", err)
					log.Fatal(err)
				}
				jsonData, err := json.Marshal(re)
				if err != nil {
					// fmt.Printf("转换为JSON失败：%v\n", err)
					log.Fatal(err)
				}
				decode_input = string(jsonData)
			}

		}

		_, err = s.db.Exec("INSERT INTO bc_block_transactions (block_num, trans_hash, `from`, `to`, input, decode_input, is_contract, method_id, import_time) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
			blockInfo.Number, tx.Hash, tx.From, tx.To, tx.Input, decode_input, is_contract, method_id, tx.ImportTime)
		if err != nil {
			if mysqlErr, ok := err.(*mysql.MySQLError); ok && mysqlErr.Number == 1062 {
				// 错误代码 1062 表示唯一性约束错误
				fmt.Printf("区块[%v]交易已存在：%v\n", blockInfo.Number, tx.Hash)
			} else {
				fmt.Printf("区块[%v]交易存储失败：%v\n", blockInfo.Number, tx.Hash)
				log.Fatal(err)
			}
		}
		s.synTransReceipt(tx.Hash)
		go s.synAddAddress(tx.From)
	}

	// 插入区块信息到数据库
	txJSON, err := json.Marshal(blockInfo.Transactions)
	if err != nil {
		log.Fatal(err)
	}
	_, err = s.db.Exec("INSERT INTO bc_block_number (block_num, block_hash, block_transactions, response_code, status) VALUES (?, ?, ?, ?, ?)",
		blockInfo.Number, blockInfo.Hash, string(txJSON), response.StatusCode, 1)
	if err != nil {
		fmt.Printf("区块存储失败[%v]\n", blockInfo.Number)
		log.Fatal(err)
	}

	fmt.Printf("区块存储成功[%v]\n", blockInfo.Number)
}

func (s *SQL) synTransReceipt(tran_hash string) {
	// 加载合约
	abi, err := abi.JSON(strings.NewReader(abiStr))
	if err != nil {
		fmt.Println("加载合约失败")
		log.Fatal(err)
	}

	// 构建请求体
	requestData := JSONRPCRequest{
		Jsonrpc: "2.0",
		Method:  "getTransactionReceipt",
		Params: []interface{}{
			"group0",
			"",
			tran_hash,
			false,
		},
		Id: 1,
	}
	requestBody, err := json.Marshal(requestData)
	if err != nil {
		fmt.Println("构建请求体失败", err)
		return
	}

	// 发送 POST 请求
	response, err := http.Post(rpcUrl, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		fmt.Println("Failed to send request:", err)
		return
	}
	defer response.Body.Close()

	// 读取响应数据
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Println("Failed to read response body:", err)
		return
	}

	// 解析响应数据
	var jsonResponse JSONRPCResponse
	err = json.Unmarshal(responseBody, &jsonResponse)
	if err != nil {
		fmt.Println("Failed to unmarshal response body:", err)
		return
	}
	var transactionReceipt TransactionReceipt
	err = json.Unmarshal(jsonResponse.Result, &transactionReceipt)
	if err != nil {
		fmt.Println("Failed to unmarshal result field:", err)
		return
	}
	// 解码output
	decode_output := ""
	method_id := ""
	if transactionReceipt.To == contractAddress {
		// 获取合约方法id
		method_id = transactionReceipt.Input[2:10]
		decodedSig, err := hex.DecodeString(method_id)
		if err != nil {
			log.Fatal(err)
		}
		if method_id == contractMethodId {
			// 加载合约方法
			method, err := abi.MethodById(decodedSig)
			if err != nil {
				log.Fatal(err)
			}
			decodedOutputData, err := hex.DecodeString(transactionReceipt.Output[2:])
			if err != nil {
				log.Fatal(err)
			}
			re, err := method.Outputs.Unpack(decodedOutputData)
			if err != nil {
				fmt.Printf("解码失败：%v\n", err)
				return
			}
			jsonData, err := json.Marshal(re)
			if err != nil {
				// fmt.Printf("转换为JSON失败：%v\n", err)
				log.Fatal(err)
			}
			decode_output = string(jsonData)
		}

	}

	// 更新信息
	rs, err := s.db.Exec("UPDATE bc_block_transactions SET output = ?, decode_output = ?, status = ?, gas_used = ? WHERE trans_hash = ?",
		transactionReceipt.Output, decode_output, transactionReceipt.Status, transactionReceipt.GasUsed, transactionReceipt.Hash)
	if err != nil {
		log.Fatal(err)
	}
	// 判断是否更新成功
	rowsAffected, err := rs.RowsAffected()
	if err != nil {
		log.Fatal(err)
	}
	if rowsAffected == 0 {
		fmt.Printf("未更新交易回执[%v]\n", transactionReceipt.Hash)
	} else {
		fmt.Printf("更新交易回执成功[%v]\n", transactionReceipt.Hash)
	}
}

func (s *SQL) synAddAddress(address string) {
	// 准备插入语句
	insertStatement := `INSERT INTO bc_block_account (address) VALUES (?)`
	stmt, err := s.db.Prepare(insertStatement)
	if err != nil {
		panic(err.Error())
	}
	defer stmt.Close()

	// 执行插入语句
	_, err = stmt.Exec(address)
	if err != nil {
		if mysqlErr, ok := err.(*mysql.MySQLError); ok && mysqlErr.Number == 1062 {
			// 错误代码 1062 表示唯一性约束错误
			// fmt.Printf("地址 %s 已存在，不在插入数据\n", address)
			return
		}
		panic(err.Error())
	}

	fmt.Printf("添加帐户地址[address]")
}

func isValidAddress(address string) bool {
	re := regexp.MustCompile("^0x[0-9a-fA-F]{40}$")
	return re.MatchString(address)
}

func writeToFile(output string) error {
	file, err := os.OpenFile("register_server.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(output)
	if err != nil {
		return err
	}

	return nil
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var input Input
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		log.Println("Failed to parse request body:", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	response := Response{}
	if !isValidAddress(input.Address) {
		response.Message = "Invalid address format"
		response.Code = 0
		jsonResponse, err := json.Marshal(response)
		if err != nil {
			log.Println("Failed to marshal JSON response:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonResponse)
		return
	}

	cmdString := fmt.Sprintf("echo call Cred %s register %s | bash ./console/start.sh", contractAddress, input.Address)
	cmd := exec.Command("bash", "-c", cmdString)
	output, err := cmd.CombinedOutput() // 获取命令的标准输出和标准错误输出
	if err != nil {
		log.Println("Failed to execute command:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if strings.Contains(string(output), "transaction status: 0") {
		response.Message = "Authorization successful"
		response.Code = 1
	} else if strings.Contains(string(output), "account already has role") {
		response.Message = "Account already authorized"
		response.Code = 1
	} else {
		response.Message = "Authorization failed"
		response.Code = 0
	}

	jsonResponse, err := json.Marshal(response)
	if err != nil {
		log.Println("Failed to marshal JSON response:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResponse)

	// 将命令执行后的结果记录到文件中
	err = writeToFile(string(cmdString))
	if err != nil {
		log.Println("Failed to write to file:", err)
	}
	err = writeToFile(string(output))
	if err != nil {
		log.Println("Failed to write to file:", err)
	}
}

func getContractAddress(w http.ResponseWriter, r *http.Request) {
	response := Response{}
	response.Data = contractAddress
	response.Message = "success"
	response.Code = 1
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		log.Println("Failed to marshal JSON response:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResponse)
	return
}

func getTransByAddress(w http.ResponseWriter, r *http.Request) {
	// 获取请求参数
	queryValues := r.URL.Query()
	pageStr := queryValues.Get("page")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page <= 0 {
		page = 1
	}
	pageSizeStr := queryValues.Get("pagesize")
	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize <= 0 || pageSize > 100 {
		pageSize = 10
	}
	// 连接数据库
	dsn := dbUsername + ":" + dbPassword + "@tcp" + "(" + dbHost + ":" + dbPort + ")/" + dbase
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "Error connecting to database")
		return
	}
	defer db.Close()

	// 计算偏移量和限制条数
	offset := (page - 1) * pageSize
	limit := pageSize

	// 执行查询
	query := "SELECT block_num, trans_hash, `from`, `to`, `status`, import_time, input, output, heart_rate, breath_rate, sleep_state, person_id, contact_name, contact_identity FROM bc_block_transactions WHERE `from` = ? ORDER BY id DESC LIMIT ?, ?"
	rows, err := db.Query(query, queryValues.Get("address"), offset, limit)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "Error querying database")
		fmt.Println("Error", err)
		return
	}
	defer rows.Close()

	// 将查询结果映射为 Transaction 数据结构的列表
	transactions := make([]TransactionResponse, 0)
	for rows.Next() {
		transaction := TransactionResponse{}
		err := rows.Scan(&transaction.BlockNumber, &transaction.Hash, &transaction.From, &transaction.To, &transaction.Status, &transaction.ImportTime, &transaction.Input, &transaction.Output, &transaction.HeartRate, &transaction.BreathRate, &transaction.SleepState, &transaction.PersonId, &transaction.ContactName, &transaction.ContactIdentity)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, "Error scanning row data")
			fmt.Println("Error", err)
			return
		}
		transactions = append(transactions, transaction)
	}
	// 获取总数
	countQuery := "SELECT COUNT(*) FROM bc_block_transactions WHERE `from` = ?"
	var total int
	err = db.QueryRow(countQuery, queryValues.Get("address")).Scan(&total)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "Error querying total count")
		fmt.Println("Error", err)
		return
	}

	// 构建 Response 结构体
	response := ResponseList{
		Data: QueryList{
			List:     transactions,
			Page:     page,
			PageSize: pageSize,
			Total:    total,
		},
		Msg:  "success",
		Code: 1,
	}

	// 序列化 Response 结构为 JSON 字符串
	responseJsonData, err := json.Marshal(response)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "Error serializing JSON data")
		return
	}

	// 设置响应头为 JSON 格式
	w.Header().Set("Content-Type", "application/json")

	// 发送响应
	w.WriteHeader(http.StatusOK)
	w.Write(responseJsonData)

}

func getResByAddress(w http.ResponseWriter, r *http.Request) {
	// 获取请求参数
	queryValues := r.URL.Query()
	pageStr := queryValues.Get("page")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page <= 0 {
		page = 1
	}
	pageSizeStr := queryValues.Get("pagesize")
	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize <= 0 || pageSize > 100 {
		pageSize = 10
	}
	// 连接数据库
	dsn := dbUsername + ":" + dbPassword + "@tcp" + "(" + dbHost + ":" + dbPort + ")/" + dbase
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "Error connecting to database")
		return
	}
	defer db.Close()

	// 计算偏移量和限制条数
	offset := (page - 1) * pageSize
	limit := pageSize

	// 执行查询

	query := "SELECT block_num, trans_hash, `from`, `to`, `status`, import_time, input, output, sleep_breathing, heart_change, person_id, contact_name, contact_identity FROM bc_block_transactions WHERE `from` = ? ORDER BY id DESC LIMIT ?, ?"
	rows, err := db.Query(query, queryValues.Get("address"), offset, limit)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "Error querying database")
		fmt.Println("Error", err)
		return
	}
	defer rows.Close()

	// 将查询结果映射为 Transaction 数据结构的列表
	transactions := make([]TransactionResResponse, 0)
	for rows.Next() {
		transaction := TransactionResResponse{}
		err := rows.Scan(&transaction.BlockNumber, &transaction.Hash, &transaction.From, &transaction.To, &transaction.Status, &transaction.ImportTime, &transaction.Input, &transaction.Output, &transaction.SleepBreathing, &transaction.HeartChange, &transaction.PersonId, &transaction.ContactName, &transaction.ContactIdentity)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, "Error scanning row data")
			fmt.Println("Error", err)
			return
		}
		transactions = append(transactions, transaction)
	}
	// 获取总数
	countQuery := "SELECT COUNT(*) FROM bc_block_transactions WHERE `from` = ?"
	var total int
	err = db.QueryRow(countQuery, queryValues.Get("address")).Scan(&total)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "Error querying total count")
		fmt.Println("Error", err)
		return
	}

	// 构建 Response 结构体
	response := ResponseList{
		Data: QueryList{
			List:     transactions,
			Page:     page,
			PageSize: pageSize,
			Total:    total,
		},
		Msg:  "success",
		Code: 1,
	}

	// 序列化 Response 结构为 JSON 字符串
	responseJsonData, err := json.Marshal(response)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "Error serializing JSON data")
		return
	}

	// 设置响应头为 JSON 格式
	w.Header().Set("Content-Type", "application/json")

	// 发送响应
	w.WriteHeader(http.StatusOK)
	w.Write(responseJsonData)

}

func accountRanking(w http.ResponseWriter, r *http.Request) {
	// 获取请求参数
	queryValues := r.URL.Query()
	pageStr := queryValues.Get("page")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page <= 0 {
		page = 1
	}
	pageSizeStr := queryValues.Get("pagesize")
	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize <= 0 || pageSize > 100 {
		pageSize = 10
	}
	// 连接数据库
	dsn := dbUsername + ":" + dbPassword + "@tcp" + "(" + dbHost + ":" + dbPort + ")/" + dbase
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "Error connecting to database")
		return
	}
	defer db.Close()

	// 计算偏移量和限制条数
	offset := (page - 1) * pageSize
	limit := pageSize

	// 执行查询
	query := "SELECT address, balance, cred, share_num FROM bc_block_account ORDER BY balance DESC LIMIT ?, ?"
	rows, err := db.Query(query, offset, limit)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "Error querying database")
		fmt.Println("Error", err)
		return
	}
	defer rows.Close()

	// 将查询结果映射为 Transaction 数据结构的列表
	accounts := make([]AccountResponse, 0)
	for rows.Next() {
		account := AccountResponse{}
		err := rows.Scan(&account.Address, &account.Balance, &account.Cred, &account.ShareNnum)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, "Error scanning row data")
			fmt.Println("Error", err)
			return
		}
		accounts = append(accounts, account)
	}
	// 获取总数
	countQuery := "SELECT COUNT(*) FROM bc_block_account"
	var total int
	err = db.QueryRow(countQuery).Scan(&total)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "Error querying total count")
		fmt.Println("Error", err)
		return
	}

	// 构建 Response 结构体
	response := ResponseList{
		Data: AccountQueryList{
			List:     accounts,
			Page:     page,
			PageSize: pageSize,
			Total:    total,
		},
		Msg:  "success",
		Code: 1,
	}

	// 序列化 Response 结构为 JSON 字符串
	responseJsonData, err := json.Marshal(response)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "Error serializing JSON data")
		return
	}

	// 设置响应头为 JSON 格式
	w.Header().Set("Content-Type", "application/json")

	// 发送响应
	w.WriteHeader(http.StatusOK)
	w.Write(responseJsonData)

}
