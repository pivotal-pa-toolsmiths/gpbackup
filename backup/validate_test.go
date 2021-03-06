package backup_test

import (
	"github.com/greenplum-db/gpbackup/backup"
	"github.com/greenplum-db/gpbackup/testutils"
	sqlmock "gopkg.in/DATA-DOG/go-sqlmock.v1"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("backup/validate tests", func() {
	var filterList []string
	AfterEach(func() {
		filterList = []string{}
	})
	Describe("ValidateFilterSchemas", func() {
		It("passes if there are no filter schemas", func() {
			backup.ValidateFilterSchemas(connection, filterList)
		})
		It("passes if single schema is present in database", func() {
			single_schema_row := sqlmock.NewRows([]string{"string"}).
				AddRow("schema1")
			mock.ExpectQuery("SELECT (.*)").WillReturnRows(single_schema_row)
			filterList = []string{"schema1"}
			backup.ValidateFilterSchemas(connection, filterList)
		})
		It("passes if multiple schemas are present in database", func() {
			two_schema_rows := sqlmock.NewRows([]string{"string"}).
				AddRow("schema1").AddRow("schema2")
			mock.ExpectQuery("SELECT (.*)").WillReturnRows(two_schema_rows)
			filterList = []string{"schema1", "schema2"}
			backup.ValidateFilterSchemas(connection, filterList)
		})
		It("panics if schema is not present in database", func() {
			two_schema_rows := sqlmock.NewRows([]string{"string"}).
				AddRow("schema1")
			mock.ExpectQuery("SELECT (.*)").WillReturnRows(two_schema_rows)
			filterList = []string{"schema1", "schema2"}
			defer testutils.ShouldPanicWithMessage("Schema schema2 does not exist")
			backup.ValidateFilterSchemas(connection, filterList)
		})
	})
	Describe("ValidateFilterTables", func() {
		var tableRows, partitionTables *sqlmock.Rows
		BeforeEach(func() {
			tableRows = sqlmock.NewRows([]string{"oid", "name"})
			partitionTables = sqlmock.NewRows([]string{"oid", "value"})
		})
		Context("Non-partition tables", func() {
			BeforeEach(func() {
				backup.SetLeafPartitionData(false)
			})
			It("passes if there are no filter tables", func() {
				backup.ValidateFilterTables(connection, filterList)
			})
			It("passes if single table is present in database", func() {
				tableRows.AddRow("1", "public.table1")
				mock.ExpectQuery("SELECT (.*)").WillReturnRows(tableRows)
				mock.ExpectQuery("SELECT (.*)").WillReturnRows(partitionTables)
				filterList = []string{"public.table1"}
				backup.ValidateFilterTables(connection, filterList)
			})
			It("passes if multiple tables are present in database", func() {
				tableRows.AddRow("1", "public.table1").AddRow("2", "public.table2")
				mock.ExpectQuery("SELECT (.*)").WillReturnRows(tableRows)
				mock.ExpectQuery("SELECT (.*)").WillReturnRows(partitionTables)
				filterList = []string{"public.table1", "public.table2"}
				backup.ValidateFilterTables(connection, filterList)
			})
			It("panics if table is not present in database", func() {
				tableRows.AddRow("1", "public.table1")
				mock.ExpectQuery("SELECT (.*)").WillReturnRows(tableRows)
				mock.ExpectQuery("SELECT (.*)").WillReturnRows(partitionTables)
				filterList = []string{"public.table1", "public.table2"}
				defer testutils.ShouldPanicWithMessage("Table public.table2 does not exist")
				backup.ValidateFilterTables(connection, filterList)
			})
		})
		Context("Partition tables", func() {
			BeforeEach(func() {
				backup.SetLeafPartitionData(false)
			})
			It("passes if given a parent partition table", func() {
				tableRows.AddRow("1", "public.table1")
				mock.ExpectQuery("SELECT (.*)").WillReturnRows(tableRows)
				partitionTables.AddRow("1", "p")
				mock.ExpectQuery("SELECT (.*)").WillReturnRows(partitionTables)
				filterList = []string{"public.table1"}
				backup.ValidateFilterTables(connection, filterList)
			})
			It("passes if given a leaf partition table", func() {
				tableRows.AddRow("1", "public.table1")
				mock.ExpectQuery("SELECT (.*)").WillReturnRows(tableRows)
				partitionTables.AddRow("1", "l")
				mock.ExpectQuery("SELECT (.*)").WillReturnRows(partitionTables)
				filterList = []string{"public.table1"}
				backup.ValidateFilterTables(connection, filterList)
			})
			It("panics if given an intermediate partition table and --leaf-partition-data is set", func() {
				backup.SetLeafPartitionData(true)
				defer backup.SetLeafPartitionData(false)
				tableRows.AddRow("1", "public.table1")
				mock.ExpectQuery("SELECT (.*)").WillReturnRows(tableRows)
				partitionTables.AddRow("1", "i")
				mock.ExpectQuery("SELECT (.*)").WillReturnRows(partitionTables)
				filterList = []string{"public.table1"}
				defer testutils.ShouldPanicWithMessage("Cannot filter on public.table1, as it is an intermediate partition table.  Only parent partition tables and leaf partition tables may be specified.")
				backup.ValidateFilterTables(connection, filterList)
			})
			It("panics if given an intermediate partition table and --leaf-partition-data is not set", func() {
				tableRows.AddRow("1", "public.table1")
				mock.ExpectQuery("SELECT (.*)").WillReturnRows(tableRows)
				partitionTables.AddRow("1", "i")
				mock.ExpectQuery("SELECT (.*)").WillReturnRows(partitionTables)
				filterList = []string{"public.table1"}
				defer testutils.ShouldPanicWithMessage("Cannot filter on public.table1, as it is an intermediate partition table.  Only parent partition tables and leaf partition tables may be specified.")
				backup.ValidateFilterTables(connection, filterList)
			})
		})
	})
	Describe("ValidateCompressionLevel", func() {
		It("validates a compression level between 1 and 9", func() {
			compressLevel := 5
			backup.ValidateCompressionLevel(compressLevel)
		})
		It("panics if given a compression level < 0", func() {
			compressLevel := -2
			defer testutils.ShouldPanicWithMessage("Compression level must be between 1 and 9")
			backup.ValidateCompressionLevel(compressLevel)
		})
		It("panics if given a compression level > 9", func() {
			compressLevel := 11
			defer testutils.ShouldPanicWithMessage("Compression level must be between 1 and 9")
			backup.ValidateCompressionLevel(compressLevel)
		})
	})
})
