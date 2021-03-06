package restore

import "github.com/greenplum-db/gpbackup/utils"

/*
 * This file contains global variables and setter functions for those variables
 * used in testing.
 */

/*
 * Non-flag variables
 */

var (
	backupConfig  *utils.BackupConfig
	connection    *utils.DBConn
	globalCluster utils.Cluster
	globalTOC     *utils.TOC
	logger        *utils.Logger
	version       string
)

/*
 * Command-line flags
 */

var (
	backupDir        *string
	createdb         *bool
	debug            *bool
	includeSchemas   utils.ArrayFlags
	includeTableFile *string
	includeTables    utils.ArrayFlags
	numJobs          *int
	onErrorContinue  *bool
	printVersion     *bool
	quiet            *bool
	redirect         *string
	restoreGlobals   *bool
	timestamp        *string
	verbose          *bool
	withStats        *bool
)

/*
 * Setter functions
 */

func SetBackupConfig(config *utils.BackupConfig) {
	backupConfig = config
}

func SetConnection(conn *utils.DBConn) {
	connection = conn
}

func SetCluster(cluster utils.Cluster) {
	globalCluster = cluster
}

func SetLogger(log *utils.Logger) {
	logger = log
}

func SetOnErrorContinue(errContinue bool) {
	onErrorContinue = &errContinue
}

func SetNumJobs(jobs int) {
	numJobs = &jobs
}

func SetTOC(toc *utils.TOC) {
	globalTOC = toc
}
