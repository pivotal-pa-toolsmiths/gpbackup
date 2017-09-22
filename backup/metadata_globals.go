package backup

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/greenplum-db/gpbackup/utils"
)

/*
 * This file contains structs and functions related to backing up global cluster
 * metadata on the master that needs to be restored before data is restored,
 * such as roles and database configuration.
 */

func PrintConnectionString(metadataFile *utils.FileWithByteCount, dbname string) {
	metadataFile.MustPrintf("\\c %s\n", dbname)
}

/*
 * Session GUCs are printed to global, predata, and postdata files so we
 * will use the correct settings when the files are run during restore
 */
func PrintPredataSessionGUCs(metadataFile *utils.FileWithByteCount, toc *utils.TOC, gucs SessionGUCs) {
	start := metadataFile.ByteCount
	printSessionGUCs(metadataFile, gucs)
	toc.AddPredataEntry("", "", "SESSION GUCS", start, metadataFile.ByteCount)
}

func PrintPostdataSessionGUCs(metadataFile *utils.FileWithByteCount, toc *utils.TOC, gucs SessionGUCs) {
	start := metadataFile.ByteCount
	printSessionGUCs(metadataFile, gucs)
	toc.AddPostdataEntry("", "", "SESSION GUCS", start, metadataFile.ByteCount)
}

func PrintGlobalSessionGUCs(metadataFile *utils.FileWithByteCount, toc *utils.TOC, gucs SessionGUCs) {
	start := metadataFile.ByteCount
	printSessionGUCs(metadataFile, gucs)
	toc.AddGlobalEntry("", "", "SESSION GUCS", start, metadataFile.ByteCount)
}

func PrintStatisticsSessionGUCs(metadataFile *utils.FileWithByteCount, toc *utils.TOC, gucs SessionGUCs) {
	start := metadataFile.ByteCount
	printSessionGUCs(metadataFile, gucs)
	toc.AddStatisticsEntry("", "", "SESSION GUCS", start, metadataFile.ByteCount)
}

func printSessionGUCs(metadataFile *utils.FileWithByteCount, gucs SessionGUCs) {
	metadataFile.MustPrintf(`SET statement_timeout = 0;
SET check_function_bodies = false;
SET client_min_messages = error;
SET client_encoding = '%s';
SET standard_conforming_strings = on;
SET default_with_oids = %s;
`, gucs.ClientEncoding, gucs.DefaultWithOids)
}

func PrintCreateDatabaseStatement(globalFile *utils.FileWithByteCount, toc *utils.TOC, dbname string, allDBs []DatabaseName, dbMetadata MetadataMap, backupGlobals bool) {
	dbname = utils.QuoteIdent(dbname)
	for _, db := range allDBs {
		if db.DatabaseName == dbname {
			start := globalFile.ByteCount
			globalFile.MustPrintf("\n\nCREATE DATABASE %s", dbname)
			if db.TablespaceName != "pg_default" {
				globalFile.MustPrintf(" TABLESPACE %s", db.TablespaceName)
			}
			globalFile.MustPrintf(";")
			toc.AddGlobalEntry("", dbname, "DATABASE", start, globalFile.ByteCount)
			start = globalFile.ByteCount
			PrintObjectMetadata(globalFile, dbMetadata[db.Oid], dbname, "DATABASE")
			if globalFile.ByteCount > start {
				toc.AddGlobalEntry("", dbname, "DATABASE METADATA", start, globalFile.ByteCount)
			}
			break
		}
	}
	if backupGlobals {
		for _, db := range allDBs {
			if db.DatabaseName != dbname {
				start := globalFile.ByteCount
				PrintObjectMetadata(globalFile, dbMetadata[db.Oid], db.DatabaseName, "DATABASE")
				toc.AddGlobalEntry("", dbname, "DATABASE OTHER", start, globalFile.ByteCount)
			}
		}
	}
}

func PrintDatabaseGUCs(globalFile *utils.FileWithByteCount, toc *utils.TOC, gucs []string, dbname string) {
	for _, guc := range gucs {
		start := globalFile.ByteCount
		globalFile.MustPrintf("\nALTER DATABASE %s %s;", utils.QuoteIdent(dbname), guc)
		toc.AddGlobalEntry("", dbname, "DATABASE GUC", start, globalFile.ByteCount)
	}
}

func PrintCreateResourceQueueStatements(globalFile *utils.FileWithByteCount, toc *utils.TOC, resQueues []ResourceQueue, resQueueMetadata MetadataMap) {
	for _, resQueue := range resQueues {
		start := globalFile.ByteCount
		attributes := []string{}
		if resQueue.ActiveStatements != -1 {
			attributes = append(attributes, fmt.Sprintf("ACTIVE_STATEMENTS=%d", resQueue.ActiveStatements))
		}
		maxCostFloat, maxCostErr := strconv.ParseFloat(resQueue.MaxCost, 64)
		utils.CheckError(maxCostErr)
		if maxCostFloat > -1 {
			attributes = append(attributes, fmt.Sprintf("MAX_COST=%s", resQueue.MaxCost))
		}
		if resQueue.CostOvercommit {
			attributes = append(attributes, "COST_OVERCOMMIT=TRUE")
		}
		minCostFloat, minCostErr := strconv.ParseFloat(resQueue.MinCost, 64)
		utils.CheckError(minCostErr)
		if minCostFloat > 0 {
			attributes = append(attributes, fmt.Sprintf("MIN_COST=%s", resQueue.MinCost))
		}
		if resQueue.Priority != "medium" {
			attributes = append(attributes, fmt.Sprintf("PRIORITY=%s", strings.ToUpper(resQueue.Priority)))
		}
		if resQueue.MemoryLimit != "-1" {
			attributes = append(attributes, fmt.Sprintf("MEMORY_LIMIT='%s'", resQueue.MemoryLimit))
		}
		action := "CREATE"
		if resQueue.Name == "pg_default" {
			action = "ALTER"
		}
		globalFile.MustPrintf("\n\n%s RESOURCE QUEUE %s WITH (%s);", action, utils.QuoteIdent(resQueue.Name), strings.Join(attributes, ", "))
		PrintObjectMetadata(globalFile, resQueueMetadata[resQueue.Oid], utils.QuoteIdent(resQueue.Name), "RESOURCE QUEUE")
		toc.AddGlobalEntry("", resQueue.Name, "RESOURCE QUEUE", start, globalFile.ByteCount)
	}
}

func PrintCreateRoleStatements(globalFile *utils.FileWithByteCount, toc *utils.TOC, roles []Role, roleMetadata MetadataMap) {
	for _, role := range roles {
		start := globalFile.ByteCount
		quotedName := utils.QuoteIdent(role.Name)
		attrs := []string{}

		if role.Super {
			attrs = append(attrs, "SUPERUSER")
		} else {
			attrs = append(attrs, "NOSUPERUSER")
		}

		if role.Inherit {
			attrs = append(attrs, "INHERIT")
		} else {
			attrs = append(attrs, "NOINHERIT")
		}

		if role.CreateRole {
			attrs = append(attrs, "CREATEROLE")
		} else {
			attrs = append(attrs, "NOCREATEROLE")
		}

		if role.CreateDB {
			attrs = append(attrs, "CREATEDB")
		} else {
			attrs = append(attrs, "NOCREATEDB")
		}

		if role.CanLogin {
			attrs = append(attrs, "LOGIN")
		} else {
			attrs = append(attrs, "NOLOGIN")
		}
		if role.ConnectionLimit != -1 {
			attrs = append(attrs, fmt.Sprintf("CONNECTION LIMIT %d", role.ConnectionLimit))
		}

		if role.Password != "" {
			attrs = append(attrs, fmt.Sprintf("PASSWORD '%s'", role.Password))
		}

		if role.ValidUntil != "" {
			attrs = append(attrs, fmt.Sprintf("VALID UNTIL '%s'", role.ValidUntil))
		}

		attrs = append(attrs, fmt.Sprintf("RESOURCE QUEUE %s", utils.QuoteIdent(role.ResQueue)))

		if role.Createrexthttp {
			attrs = append(attrs, "CREATEEXTTABLE (protocol='http')")
		}

		if role.Createrextgpfd {
			attrs = append(attrs, "CREATEEXTTABLE (protocol='gpfdist', type='readable')")
		}

		if role.Createwextgpfd {
			attrs = append(attrs, "CREATEEXTTABLE (protocol='gpfdist', type='writable')")
		}

		if role.Createrexthdfs {
			attrs = append(attrs, "CREATEEXTTABLE (protocol='gphdfs', type='readable')")
		}

		if role.Createwexthdfs {
			attrs = append(attrs, "CREATEEXTTABLE (protocol='gphdfs', type='writable')")
		}

		globalFile.MustPrintf(`

CREATE ROLE %s;
ALTER ROLE %s WITH %s;`, quotedName, quotedName, strings.Join(attrs, " "))

		if len(role.TimeConstraints) != 0 {
			for _, timeConstraint := range role.TimeConstraints {
				globalFile.MustPrintf("\nALTER ROLE %s DENY BETWEEN DAY %d TIME '%s' AND DAY %d TIME '%s';", quotedName, timeConstraint.StartDay, timeConstraint.StartTime, timeConstraint.EndDay, timeConstraint.EndTime)
			}
		}
		PrintObjectMetadata(globalFile, roleMetadata[role.Oid], quotedName, "ROLE")
		toc.AddGlobalEntry("", role.Name, "ROLE", start, globalFile.ByteCount)
	}
}

func PrintRoleMembershipStatements(globalFile *utils.FileWithByteCount, toc *utils.TOC, roleMembers []RoleMember) {
	globalFile.MustPrintln("\n")
	for _, roleMember := range roleMembers {
		start := globalFile.ByteCount
		globalFile.MustPrintf("\nGRANT %s TO %s", roleMember.Role, roleMember.Member)
		if roleMember.IsAdmin {
			globalFile.MustPrintf(" WITH ADMIN OPTION")
		}
		globalFile.MustPrintf(" GRANTED BY %s;", roleMember.Grantor)
		toc.AddGlobalEntry("", roleMember.Member, "ROLE GRANT", start, globalFile.ByteCount)
	}
}

func PrintCreateTablespaceStatements(globalFile *utils.FileWithByteCount, toc *utils.TOC, tablespaces []Tablespace, tablespaceMetadata MetadataMap) {
	for _, tablespace := range tablespaces {
		start := globalFile.ByteCount
		globalFile.MustPrintf("\n\nCREATE TABLESPACE %s FILESPACE %s;", tablespace.Tablespace, tablespace.Filespace)
		PrintObjectMetadata(globalFile, tablespaceMetadata[tablespace.Oid], tablespace.Tablespace, "TABLESPACE")
		toc.AddGlobalEntry("", tablespace.Tablespace, "TABLESPACE", start, globalFile.ByteCount)
	}
}
