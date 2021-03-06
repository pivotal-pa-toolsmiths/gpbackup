package integration

import (
	"sort"

	"github.com/greenplum-db/gpbackup/backup"
	"github.com/greenplum-db/gpbackup/testutils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("backup integration create statement tests", func() {
	BeforeEach(func() {
		toc, backupfile = testutils.InitializeTestTOC(buffer, "predata")
	})
	Describe("PrintRegularTableCreateStatement", func() {
		var (
			extTableEmpty backup.ExternalTableDefinition
			testTable     backup.Relation
			tableDef      backup.TableDefinition
			tableDefs     map[uint32]backup.TableDefinition
			/*
			 * We need to construct partitionDef and partTemplateDef piecemeal like this,
			 * or go fmt will remove the trailing whitespace and prevent literal comparison.
			 */
			partitionDef = `PARTITION BY LIST(gender) ` + `
          (
          PARTITION girls VALUES('F') WITH (tablename='rank_1_prt_girls', appendonly=false ), ` + `
          PARTITION boys VALUES('M') WITH (tablename='rank_1_prt_boys', appendonly=false ), ` + `
          DEFAULT PARTITION other  WITH (tablename='rank_1_prt_other', appendonly=false )
          )`
			subpartitionDef = `PARTITION BY LIST(gender)
          SUBPARTITION BY LIST(region) ` + `
          (
          PARTITION girls VALUES('F') WITH (tablename='rank_1_prt_girls', appendonly=false ) ` + `
                  (
                  SUBPARTITION usa VALUES('usa') WITH (tablename='rank_1_prt_girls_2_prt_usa', appendonly=false ), ` + `
                  SUBPARTITION asia VALUES('asia') WITH (tablename='rank_1_prt_girls_2_prt_asia', appendonly=false ), ` + `
                  SUBPARTITION europe VALUES('europe') WITH (tablename='rank_1_prt_girls_2_prt_europe', appendonly=false ), ` + `
                  DEFAULT SUBPARTITION other_regions  WITH (tablename='rank_1_prt_girls_2_prt_other_regions', appendonly=false )
                  ), ` + `
          PARTITION boys VALUES('M') WITH (tablename='rank_1_prt_boys', appendonly=false ) ` + `
                  (
                  SUBPARTITION usa VALUES('usa') WITH (tablename='rank_1_prt_boys_2_prt_usa', appendonly=false ), ` + `
                  SUBPARTITION asia VALUES('asia') WITH (tablename='rank_1_prt_boys_2_prt_asia', appendonly=false ), ` + `
                  SUBPARTITION europe VALUES('europe') WITH (tablename='rank_1_prt_boys_2_prt_europe', appendonly=false ), ` + `
                  DEFAULT SUBPARTITION other_regions  WITH (tablename='rank_1_prt_boys_2_prt_other_regions', appendonly=false )
                  ), ` + `
          DEFAULT PARTITION other  WITH (tablename='rank_1_prt_other', appendonly=false ) ` + `
                  (
                  SUBPARTITION usa VALUES('usa') WITH (tablename='rank_1_prt_other_2_prt_usa', appendonly=false ), ` + `
                  SUBPARTITION asia VALUES('asia') WITH (tablename='rank_1_prt_other_2_prt_asia', appendonly=false ), ` + `
                  SUBPARTITION europe VALUES('europe') WITH (tablename='rank_1_prt_other_2_prt_europe', appendonly=false ), ` + `
                  DEFAULT SUBPARTITION other_regions  WITH (tablename='rank_1_prt_other_2_prt_other_regions', appendonly=false )
                  )
          )`
			partTemplateDef = `ALTER TABLE testtable ` + `
SET SUBPARTITION TEMPLATE  ` + `
          (
          SUBPARTITION usa VALUES('usa') WITH (tablename='testtable'), ` + `
          SUBPARTITION asia VALUES('asia') WITH (tablename='testtable'), ` + `
          SUBPARTITION europe VALUES('europe') WITH (tablename='testtable'), ` + `
          DEFAULT SUBPARTITION other_regions  WITH (tablename='testtable')
          )
`
		)
		BeforeEach(func() {
			extTableEmpty = backup.ExternalTableDefinition{Oid: 0, Type: -2, Protocol: -2, Location: "", ExecLocation: "ALL_SEGMENTS", FormatType: "t", FormatOpts: "", Options: "", Command: "", RejectLimit: 0, RejectLimitType: "", ErrTable: "", Encoding: "UTF-8", Writable: false, URIs: nil}
			testTable = backup.BasicRelation("public", "testtable")
			tableDef = backup.TableDefinition{DistPolicy: "DISTRIBUTED RANDOMLY", ExtTableDef: extTableEmpty}
			tableDefs = map[uint32]backup.TableDefinition{}
		})
		AfterEach(func() {
			testTable.DependsUpon = []string{}
			testTable.Inherits = []string{}
			testutils.AssertQueryRuns(connection, "DROP TABLE IF EXISTS public.testtable")
		})
		It("creates a table with no attributes", func() {
			backup.PrintRegularTableCreateStatement(backupfile, toc, testTable, tableDef)

			testutils.AssertQueryRuns(connection, buffer.String())
			testTable.Oid = testutils.OidFromObjectName(connection, "public", "testtable", backup.TYPE_RELATION)
			resultTableDef := backup.ConstructDefinitionsForTables(connection, []backup.Relation{testTable})[testTable.Oid]
			testutils.ExpectStructsToMatchExcluding(&tableDef, &resultTableDef, "ColumnDefs.Oid", "ExtTableDef")
		})
		It("creates a basic heap table", func() {
			rowOne := backup.ColumnDefinition{Oid: 0, Num: 1, Name: "i", NotNull: false, HasDefault: false, Type: "integer", Encoding: "", StatTarget: -1, StorageType: "", DefaultVal: "", Comment: ""}
			rowTwo := backup.ColumnDefinition{Oid: 0, Num: 2, Name: "j", NotNull: false, HasDefault: false, Type: "character varying(20)", Encoding: "", StatTarget: -1, StorageType: "", DefaultVal: "", Comment: ""}
			tableDef.ColumnDefs = []backup.ColumnDefinition{rowOne, rowTwo}

			backup.PrintRegularTableCreateStatement(backupfile, toc, testTable, tableDef)

			testutils.AssertQueryRuns(connection, buffer.String())
			testTable.Oid = testutils.OidFromObjectName(connection, "public", "testtable", backup.TYPE_RELATION)
			resultTableDef := backup.ConstructDefinitionsForTables(connection, []backup.Relation{testTable})[testTable.Oid]
			testutils.ExpectStructsToMatchExcluding(&tableDef, &resultTableDef, "ColumnDefs.Oid", "ExtTableDef")
		})
		It("creates a complex heap table", func() {
			rowOneDefault := backup.ColumnDefinition{Oid: 0, Num: 1, Name: "i", NotNull: false, HasDefault: true, Type: "integer", Encoding: "", StatTarget: -1, StorageType: "", DefaultVal: "42", Comment: ""}
			rowNotNullDefault := backup.ColumnDefinition{Oid: 0, Num: 2, Name: "j", NotNull: true, HasDefault: true, Type: "character varying(20)", Encoding: "", StatTarget: -1, StorageType: "", DefaultVal: "'bar'::text", Comment: ""}
			rowNonDefaultStorageAndStats := backup.ColumnDefinition{Oid: 0, Num: 3, Name: "k", NotNull: false, HasDefault: false, Type: "text", Encoding: "", StatTarget: 3, StorageType: "PLAIN", DefaultVal: "", Comment: ""}
			tableDef.DistPolicy = "DISTRIBUTED BY (i, j)"
			tableDef.ColumnDefs = []backup.ColumnDefinition{rowOneDefault, rowNotNullDefault, rowNonDefaultStorageAndStats}

			backup.PrintRegularTableCreateStatement(backupfile, toc, testTable, tableDef)

			testutils.AssertQueryRuns(connection, buffer.String())
			testTable.Oid = testutils.OidFromObjectName(connection, "public", "testtable", backup.TYPE_RELATION)
			resultTableDef := backup.ConstructDefinitionsForTables(connection, []backup.Relation{testTable})[testTable.Oid]
			testutils.ExpectStructsToMatchExcluding(&tableDef, &resultTableDef, "ColumnDefs.Oid", "ExtTableDef")
		})
		It("creates a basic append-optimized column-oriented table", func() {
			rowOne := backup.ColumnDefinition{Oid: 0, Num: 1, Name: "i", NotNull: false, HasDefault: false, Type: "integer", Encoding: "compresstype=zlib,blocksize=32768,compresslevel=1", StatTarget: -1, StorageType: "", DefaultVal: "", Comment: ""}
			rowTwo := backup.ColumnDefinition{Oid: 0, Num: 2, Name: "j", NotNull: false, HasDefault: false, Type: "character varying(20)", Encoding: "compresstype=zlib,blocksize=32768,compresslevel=1", StatTarget: -1, StorageType: "", DefaultVal: "", Comment: ""}
			tableDef.StorageOpts = "appendonly=true, orientation=column, fillfactor=42, compresstype=zlib, blocksize=32768, compresslevel=1"
			tableDef.ColumnDefs = []backup.ColumnDefinition{rowOne, rowTwo}

			backup.PrintRegularTableCreateStatement(backupfile, toc, testTable, tableDef)

			testutils.AssertQueryRuns(connection, buffer.String())
			testTable.Oid = testutils.OidFromObjectName(connection, "public", "testtable", backup.TYPE_RELATION)
			resultTableDef := backup.ConstructDefinitionsForTables(connection, []backup.Relation{testTable})[testTable.Oid]
			testutils.ExpectStructsToMatchExcluding(&tableDef, &resultTableDef, "ColumnDefs.Oid", "ExtTableDef")
		})
		It("creates a one-level partition table", func() {
			rowOne := backup.ColumnDefinition{Oid: 0, Num: 1, Name: "region", NotNull: false, HasDefault: false, Type: "text", Encoding: "", StatTarget: -1, StorageType: "", DefaultVal: "", Comment: ""}
			rowTwo := backup.ColumnDefinition{Oid: 0, Num: 2, Name: "gender", NotNull: false, HasDefault: false, Type: "text", Encoding: "", StatTarget: -1, StorageType: "", DefaultVal: "", Comment: ""}
			tableDef.PartDef = partitionDef
			tableDef.ColumnDefs = []backup.ColumnDefinition{rowOne, rowTwo}
			tableDef.PartitionType = "p"

			backup.PrintRegularTableCreateStatement(backupfile, toc, testTable, tableDef)

			testutils.AssertQueryRuns(connection, buffer.String())
			testTable.Oid = testutils.OidFromObjectName(connection, "public", "testtable", backup.TYPE_RELATION)
			resultTableDef := backup.ConstructDefinitionsForTables(connection, []backup.Relation{testTable})[testTable.Oid]
			testutils.ExpectStructsToMatchExcluding(&tableDef, &resultTableDef, "ColumnDefs.Oid", "ExtTableDef")
		})
		It("creates a two-level partition table", func() {
			rowOne := backup.ColumnDefinition{Oid: 0, Num: 1, Name: "region", NotNull: false, HasDefault: false, Type: "text", Encoding: "", StatTarget: -1, StorageType: "", DefaultVal: "", Comment: ""}
			rowTwo := backup.ColumnDefinition{Oid: 0, Num: 2, Name: "gender", NotNull: false, HasDefault: false, Type: "text", Encoding: "", StatTarget: -1, StorageType: "", DefaultVal: "", Comment: ""}
			tableDef.PartDef = subpartitionDef
			tableDef.PartTemplateDef = partTemplateDef
			tableDef.ColumnDefs = []backup.ColumnDefinition{rowOne, rowTwo}
			tableDef.PartitionType = "p"

			backup.PrintRegularTableCreateStatement(backupfile, toc, testTable, tableDef)

			testutils.AssertQueryRuns(connection, buffer.String())
			testTable.Oid = testutils.OidFromObjectName(connection, "public", "testtable", backup.TYPE_RELATION)
			resultTableDef := backup.ConstructDefinitionsForTables(connection, []backup.Relation{testTable})[testTable.Oid]
			testutils.ExpectStructsToMatchExcluding(&tableDef, &resultTableDef, "ColumnDefs.Oid", "ExtTableDef")
		})
		It("creates a table with a non-default tablespace", func() {
			testTable = backup.BasicRelation("public", "testtable2")
			testutils.AssertQueryRuns(connection, "CREATE TABLESPACE test_tablespace FILESPACE test_filespace")
			defer testutils.AssertQueryRuns(connection, "DROP TABLESPACE test_tablespace")
			tableDef.TablespaceName = "test_tablespace"

			backup.PrintRegularTableCreateStatement(backupfile, toc, testTable, tableDef)
			defer testutils.AssertQueryRuns(connection, "DROP TABLE testtable2")

			testutils.AssertQueryRuns(connection, buffer.String())
			testTable.Oid = testutils.OidFromObjectName(connection, "public", "testtable2", backup.TYPE_RELATION)
			resultTableDef := backup.ConstructDefinitionsForTables(connection, []backup.Relation{testTable})[testTable.Oid]
			testutils.ExpectStructsToMatchExcluding(&tableDef, &resultTableDef, "ColumnDefs.Oid", "ExtTableDef")
		})
		It("creates a table that inherits from one table", func() {
			testutils.AssertQueryRuns(connection, "CREATE TABLE public.parent (i int)")
			defer testutils.AssertQueryRuns(connection, "DROP TABLE public.parent")
			tableDef.ColumnDefs = []backup.ColumnDefinition{}
			testTable.DependsUpon = []string{"public.parent"}
			testTable.Inherits = []string{"public.parent"}

			backup.PrintRegularTableCreateStatement(backupfile, toc, testTable, tableDef)

			testutils.AssertQueryRuns(connection, buffer.String())
			defer testutils.AssertQueryRuns(connection, "DROP TABLE public.testtable")
			testTable.Oid = testutils.OidFromObjectName(connection, "public", "testtable", backup.TYPE_RELATION)
			tables := []backup.Relation{testTable}
			tables = backup.ConstructTableDependencies(connection, tables, tableDefs, false)

			Expect(len(tables)).To(Equal(1))
			Expect(len(tables[0].DependsUpon)).To(Equal(1))
			Expect(len(tables[0].Inherits)).To(Equal(1))
			Expect(tables[0].DependsUpon[0]).To(Equal("public.parent"))
			Expect(tables[0].Inherits[0]).To(Equal("public.parent"))
		})
		It("creates a table that inherits from two tables", func() {
			testutils.AssertQueryRuns(connection, "CREATE TABLE public.parent_one (i int)")
			defer testutils.AssertQueryRuns(connection, "DROP TABLE public.parent_one")
			testutils.AssertQueryRuns(connection, "CREATE TABLE public.parent_two (j character varying(20))")
			defer testutils.AssertQueryRuns(connection, "DROP TABLE public.parent_two")
			tableDef.ColumnDefs = []backup.ColumnDefinition{}
			testTable.DependsUpon = []string{"public.parent_one", "public.parent_two"}
			testTable.Inherits = []string{"public.parent_one", "public.parent_two"}

			backup.PrintRegularTableCreateStatement(backupfile, toc, testTable, tableDef)

			testutils.AssertQueryRuns(connection, buffer.String())
			defer testutils.AssertQueryRuns(connection, "DROP TABLE public.testtable")
			testTable.Oid = testutils.OidFromObjectName(connection, "public", "testtable", backup.TYPE_RELATION)
			tables := []backup.Relation{testTable}
			tables = backup.ConstructTableDependencies(connection, tables, tableDefs, false)

			sort.Strings(tables[0].DependsUpon)
			sort.Strings(tables[0].Inherits)
			Expect(len(tables)).To(Equal(1))
			Expect(len(tables[0].DependsUpon)).To(Equal(2))
			Expect(tables[0].DependsUpon[0]).To(Equal("public.parent_one"))
			Expect(tables[0].DependsUpon[1]).To(Equal("public.parent_two"))
			Expect(len(tables[0].Inherits)).To(Equal(2))
			Expect(tables[0].Inherits[0]).To(Equal("public.parent_one"))
			Expect(tables[0].Inherits[1]).To(Equal("public.parent_two"))
		})
	})
	Describe("PrintPostCreateTableStatements", func() {
		var (
			extTableEmpty = backup.ExternalTableDefinition{Oid: 0, Type: -2, Protocol: -2, Location: "", ExecLocation: "ALL_SEGMENTS", FormatType: "t", FormatOpts: "", Options: "", Command: "", RejectLimit: 0, RejectLimitType: "", ErrTable: "", Encoding: "UTF-8", Writable: false, URIs: nil}
			testTable     = backup.BasicRelation("public", "testtable")
			tableRow      = backup.ColumnDefinition{Oid: 0, Num: 1, Name: "i", NotNull: false, HasDefault: false, Type: "integer", Encoding: "", StatTarget: -1, StorageType: "", DefaultVal: "", Comment: ""}
			tableDef      = backup.TableDefinition{DistPolicy: "DISTRIBUTED BY (i)", ColumnDefs: []backup.ColumnDefinition{tableRow}, ExtTableDef: extTableEmpty}
			tableMetadata backup.ObjectMetadata
		)
		BeforeEach(func() {
			testutils.AssertQueryRuns(connection, "CREATE TABLE testtable(i int)")
			tableMetadata = backup.ObjectMetadata{Privileges: []backup.ACL{}}
		})
		AfterEach(func() {
			testutils.AssertQueryRuns(connection, "DROP TABLE testtable")
		})
		It("prints only owner for a table with no comment or column comments", func() {
			tableMetadata.Owner = "testrole"
			backup.PrintPostCreateTableStatements(backupfile, testTable, tableDef, tableMetadata)

			testutils.AssertQueryRuns(connection, buffer.String())
			testTable.Oid = testutils.OidFromObjectName(connection, "public", "testtable", backup.TYPE_RELATION)
			resultMetadata := backup.GetMetadataForObjectType(connection, backup.TYPE_RELATION)
			resultTableMetadata := resultMetadata[testTable.Oid]
			testutils.ExpectStructsToMatch(&tableMetadata, &resultTableMetadata)
			resultTableDef := backup.ConstructDefinitionsForTables(connection, []backup.Relation{testTable})[testTable.Oid]
			testutils.ExpectStructsToMatchExcluding(&tableDef, &resultTableDef, "ColumnDefs.Oid", "ExtTableDef")
		})
		It("prints table comment, table owner, and column comments for a table with all three", func() {
			tableMetadata.Owner = "testrole"
			tableMetadata.Comment = "This is a table comment."
			tableDef.ColumnDefs[0].Comment = "This is a column comment."
			backup.PrintPostCreateTableStatements(backupfile, testTable, tableDef, tableMetadata)

			testutils.AssertQueryRuns(connection, buffer.String())
			testTable.Oid = testutils.OidFromObjectName(connection, "public", "testtable", backup.TYPE_RELATION)
			resultTableDef := backup.ConstructDefinitionsForTables(connection, []backup.Relation{testTable})[testTable.Oid]
			testutils.ExpectStructsToMatchExcluding(&tableDef, &resultTableDef, "ColumnDefs.Oid", "ExtTableDef")
			resultMetadata := backup.GetMetadataForObjectType(connection, backup.TYPE_RELATION)
			resultTableMetadata := resultMetadata[testTable.Oid]
			testutils.ExpectStructsToMatch(&tableMetadata, &resultTableMetadata)
		})
	})
	Describe("PrintCreateViewStatements", func() {
		It("creates a view with privileges and a comment (can't specify owner in GPDB5)", func() {
			viewDef := backup.View{Oid: 1, Schema: "public", Name: "simpleview", Definition: "SELECT pg_roles.rolname FROM pg_roles;", DependsUpon: nil}
			viewMetadataMap := testutils.DefaultMetadataMap("VIEW", true, true, true)
			viewMetadata := viewMetadataMap[1]

			backup.PrintCreateViewStatements(backupfile, toc, []backup.View{viewDef}, viewMetadataMap)

			testutils.AssertQueryRuns(connection, buffer.String())
			defer testutils.AssertQueryRuns(connection, "DROP VIEW simpleview")

			resultViews := backup.GetViews(connection)
			resultMetadataMap := backup.GetMetadataForObjectType(connection, backup.TYPE_RELATION)

			viewDef.Oid = testutils.OidFromObjectName(connection, "public", "simpleview", backup.TYPE_RELATION)
			Expect(len(resultViews)).To(Equal(1))
			resultMetadata := resultMetadataMap[viewDef.Oid]
			testutils.ExpectStructsToMatch(&viewDef, &resultViews[0])
			testutils.ExpectStructsToMatch(&viewMetadata, &resultMetadata)
		})
	})
	Describe("PrintCreateSequenceStatements", func() {
		var (
			sequence            backup.Relation
			sequenceDef         backup.Sequence
			sequenceMetadataMap backup.MetadataMap
		)
		BeforeEach(func() {
			sequence = backup.Relation{SchemaOid: 0, Oid: 1, Schema: "public", Name: "my_sequence", DependsUpon: nil, Inherits: nil}
			sequenceDef = backup.Sequence{Relation: sequence}
			sequenceMetadataMap = backup.MetadataMap{}
		})
		It("creates a basic sequence", func() {
			startValue := int64(0)
			if connection.Version.AtLeast("6") {
				startValue = 1
			}
			sequenceDef.SequenceDefinition = backup.SequenceDefinition{Name: "my_sequence", LastVal: 1, Increment: 1, MaxVal: 9223372036854775807, MinVal: 1, CacheVal: 1, StartVal: startValue}
			backup.PrintCreateSequenceStatements(backupfile, toc, []backup.Sequence{sequenceDef}, sequenceMetadataMap)
			if connection.Version.Before("5") {
				sequenceDef.LogCnt = 1 // In GPDB 4.3, sequence log count is one-indexed
			}

			testutils.AssertQueryRuns(connection, buffer.String())
			defer testutils.AssertQueryRuns(connection, "DROP SEQUENCE my_sequence")

			resultSequences := backup.GetAllSequences(connection)

			Expect(len(resultSequences)).To(Equal(1))
			testutils.ExpectStructsToMatchExcluding(&sequence, &resultSequences[0].Relation, "SchemaOid", "Oid")
			testutils.ExpectStructsToMatch(&sequenceDef.SequenceDefinition, &resultSequences[0].SequenceDefinition)
		})
		It("creates a complex sequence", func() {
			startValue := int64(0)
			if connection.Version.AtLeast("6") {
				startValue = 105
			}
			sequenceDef.SequenceDefinition = backup.SequenceDefinition{Name: "my_sequence", LastVal: 105, Increment: 5, MaxVal: 1000, MinVal: 20, CacheVal: 1, LogCnt: 0, IsCycled: false, IsCalled: true, StartVal: startValue}
			backup.PrintCreateSequenceStatements(backupfile, toc, []backup.Sequence{sequenceDef}, sequenceMetadataMap)

			testutils.AssertQueryRuns(connection, buffer.String())
			defer testutils.AssertQueryRuns(connection, "DROP SEQUENCE my_sequence")

			resultSequences := backup.GetAllSequences(connection)

			Expect(len(resultSequences)).To(Equal(1))
			testutils.ExpectStructsToMatchExcluding(&sequence, &resultSequences[0].Relation, "SchemaOid", "Oid")
			testutils.ExpectStructsToMatch(&sequenceDef.SequenceDefinition, &resultSequences[0].SequenceDefinition)
		})
		It("creates a sequence with privileges, owner, and comment", func() {
			startValue := int64(0)
			if connection.Version.AtLeast("6") {
				startValue = 1
			}
			sequenceDef.SequenceDefinition = backup.SequenceDefinition{Name: "my_sequence", LastVal: 1, Increment: 1, MaxVal: 9223372036854775807, MinVal: 1, CacheVal: 1, StartVal: startValue}
			sequenceMetadata := backup.ObjectMetadata{Privileges: []backup.ACL{testutils.DefaultACLWithout("testrole", "SEQUENCE", "UPDATE")}, Owner: "testrole", Comment: "This is a sequence comment."}
			sequenceMetadataMap[1] = sequenceMetadata
			backup.PrintCreateSequenceStatements(backupfile, toc, []backup.Sequence{sequenceDef}, sequenceMetadataMap)
			if connection.Version.Before("5") {
				sequenceDef.LogCnt = 1 // In GPDB 4.3, sequence log count is one-indexed
			}

			testutils.AssertQueryRuns(connection, buffer.String())
			defer testutils.AssertQueryRuns(connection, "DROP SEQUENCE my_sequence")

			resultSequences := backup.GetAllSequences(connection)

			Expect(len(resultSequences)).To(Equal(1))
			resultMetadataMap := backup.GetMetadataForObjectType(connection, backup.TYPE_RELATION)
			oid := testutils.OidFromObjectName(connection, "public", "my_sequence", backup.TYPE_RELATION)
			resultMetadata := resultMetadataMap[oid]
			testutils.ExpectStructsToMatchExcluding(&sequence, &resultSequences[0].Relation, "SchemaOid", "Oid")
			testutils.ExpectStructsToMatch(&sequenceDef.SequenceDefinition, &resultSequences[0].SequenceDefinition)
			testutils.ExpectStructsToMatch(&sequenceMetadata, &resultMetadata)
		})
	})
	Describe("PrintAlterSequenceStatements", func() {
		It("creates a sequence owned by a table column", func() {
			startValue := int64(0)
			if connection.Version.AtLeast("6") {
				startValue = 1
			}
			sequenceDef := backup.Sequence{Relation: backup.Relation{SchemaOid: 0, Oid: 1, Schema: "public", Name: "my_sequence", DependsUpon: nil, Inherits: nil}}
			columnOwnerMap := map[string]string{"public.my_sequence": "public.sequence_table.a"}

			sequenceDef.SequenceDefinition = backup.SequenceDefinition{Name: "my_sequence",
				LastVal: 1, Increment: 1, MaxVal: 9223372036854775807, MinVal: 1, CacheVal: 1, StartVal: startValue}
			backup.PrintCreateSequenceStatements(backupfile, toc, []backup.Sequence{sequenceDef}, backup.MetadataMap{})
			backup.PrintAlterSequenceStatements(backupfile, toc, []backup.Sequence{sequenceDef}, columnOwnerMap)

			//Create table that sequence can be owned by
			testutils.AssertQueryRuns(connection, "CREATE TABLE sequence_table(a int)")
			defer testutils.AssertQueryRuns(connection, "DROP TABLE sequence_table")

			testutils.AssertQueryRuns(connection, buffer.String())
			defer testutils.AssertQueryRuns(connection, "DROP SEQUENCE my_sequence")

			sequenceOwners := backup.GetSequenceColumnOwnerMap(connection)
			Expect(len(sequenceOwners)).To(Equal(1))
			Expect(sequenceOwners["public.my_sequence"]).To(Equal("public.sequence_table.a"))
		})
	})
})
