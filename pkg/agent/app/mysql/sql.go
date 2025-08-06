package mysql

const (
	checkCloneAvaliableSql = `SELECT PLUGIN_STATUS FROM INFORMATION_SCHEMA.PLUGINS  WHERE PLUGIN_NAME = 'clone';`
	SetValidDonorListSql   = `SET GLOBAL clone_valid_donor_list = ?;`
	ExecCloneSql           = `CLONE INSTANCE FROM %s@'%s':%d IDENTIFIED BY '%s';`
	getCloneStatusSql      = `SELECT STATE, ERROR_MESSAGE FROM performance_schema.clone_status;`
	setVariableSql         = `SET GLOBAL %s = %s;`
)
