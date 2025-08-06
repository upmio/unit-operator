package proxysql

const (
	setVariableSql       = `SET %s-%s = %s;`
	loadMysqlVariableSql = `LOAD MYSQL VARIABLES TO RUNTIME`
	loadAdminVariableSql = `LOAD ADMIN VARIABLES TO RUNTIME`
)
