package integration

import (
	"github.com/greenplum-db/gpbackup/backup"
	"github.com/greenplum-db/gpbackup/testutils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("backup integration create statement tests", func() {
	BeforeEach(func() {
		toc, backupfile = testutils.InitializeTestTOC(buffer, "predata")
	})
	Describe("PrintCreateOperatorStatements", func() {
		It("creates operator", func() {
			testutils.AssertQueryRuns(connection, "CREATE SCHEMA testschema")
			defer testutils.AssertQueryRuns(connection, "DROP SCHEMA testschema")

			testutils.AssertQueryRuns(connection, "CREATE FUNCTION testschema.\"testFunc\" (path,path) RETURNS path AS 'SELECT $1' LANGUAGE SQL IMMUTABLE")
			defer testutils.AssertQueryRuns(connection, "DROP FUNCTION testschema.\"testFunc\" (path,path)")

			operator := backup.Operator{Oid: 0, Schema: "testschema", Name: "##", Procedure: "testschema.\"testFunc\"", LeftArgType: "path", RightArgType: "path", CommutatorOp: "0", NegatorOp: "0", RestrictFunction: "-", JoinFunction: "-", CanHash: false, CanMerge: false}
			operators := []backup.Operator{operator}

			backup.PrintCreateOperatorStatements(backupfile, toc, operators, backup.MetadataMap{})

			testutils.AssertQueryRuns(connection, buffer.String())
			defer testutils.AssertQueryRuns(connection, "DROP OPERATOR testschema.##(path, path)")

			resultOperators := backup.GetOperators(connection)
			Expect(len(resultOperators)).To(Equal(1))
			testutils.ExpectStructsToMatchExcluding(&operator, &resultOperators[0], "Oid")
		})
		It("creates operator with owner and comment", func() {
			testutils.AssertQueryRuns(connection, "CREATE SCHEMA testschema")
			defer testutils.AssertQueryRuns(connection, "DROP SCHEMA testschema")

			testutils.AssertQueryRuns(connection, "CREATE FUNCTION testschema.\"testFunc\" (path,path) RETURNS path AS 'SELECT $1' LANGUAGE SQL IMMUTABLE")
			defer testutils.AssertQueryRuns(connection, "DROP FUNCTION testschema.\"testFunc\" (path,path)")

			operatorMetadataMap := testutils.DefaultMetadataMap("OPERATOR", false, false, true)
			operatorMetadata := operatorMetadataMap[1]
			operator := backup.Operator{Oid: 1, Schema: "testschema", Name: "##", Procedure: "testschema.\"testFunc\"", LeftArgType: "path", RightArgType: "path", CommutatorOp: "0", NegatorOp: "0", RestrictFunction: "-", JoinFunction: "-", CanHash: false, CanMerge: false}
			operators := []backup.Operator{operator}

			backup.PrintCreateOperatorStatements(backupfile, toc, operators, operatorMetadataMap)
			testutils.AssertQueryRuns(connection, buffer.String())
			defer testutils.AssertQueryRuns(connection, "DROP OPERATOR testschema.##(path, path)")

			resultOperators := backup.GetOperators(connection)
			Expect(len(resultOperators)).To(Equal(1))
			resultMetadataMap := backup.GetCommentsForObjectType(connection, backup.TYPE_OPERATOR)
			resultMetadata := resultMetadataMap[resultOperators[0].Oid]
			testutils.ExpectStructsToMatchExcluding(&operator, &resultOperators[0], "Oid")
			testutils.ExpectStructsToMatchExcluding(&resultMetadata, &operatorMetadata, "Oid")
		})
	})
	Describe("PrintCreateOperatorFamilyStatements", func() {
		BeforeEach(func() {
			testutils.SkipIf4(connection)
		})
		It("creates operator family", func() {
			operatorFamily := backup.OperatorFamily{Oid: 1, Schema: "public", Name: "testfam", IndexMethod: "hash"}
			operatorFamilies := []backup.OperatorFamily{operatorFamily}

			backup.PrintCreateOperatorFamilyStatements(backupfile, toc, operatorFamilies, backup.MetadataMap{})

			testutils.AssertQueryRuns(connection, buffer.String())
			defer testutils.AssertQueryRuns(connection, "DROP OPERATOR FAMILY public.testfam USING hash")

			resultOperatorFamilies := backup.GetOperatorFamilies(connection)
			Expect(len(resultOperatorFamilies)).To(Equal(1))
			testutils.ExpectStructsToMatchExcluding(&operatorFamily, &resultOperatorFamilies[0], "Oid")
		})
		It("creates operator family with owner and comment", func() {
			operatorFamilyMetadataMap := testutils.DefaultMetadataMap("OPERATOR FAMILY", false, true, true)
			operatorFamilyMetadata := operatorFamilyMetadataMap[1]
			operatorFamily := backup.OperatorFamily{Oid: 1, Schema: "public", Name: "testfam", IndexMethod: "hash"}
			operatorFamilies := []backup.OperatorFamily{operatorFamily}

			backup.PrintCreateOperatorFamilyStatements(backupfile, toc, operatorFamilies, operatorFamilyMetadataMap)

			testutils.AssertQueryRuns(connection, buffer.String())
			defer testutils.AssertQueryRuns(connection, "DROP OPERATOR FAMILY public.testfam USING hash")

			resultOperatorFamilies := backup.GetOperatorFamilies(connection)
			Expect(len(resultOperatorFamilies)).To(Equal(1))
			resultMetadataMap := backup.GetMetadataForObjectType(connection, backup.TYPE_OPERATORFAMILY)
			resultMetadata := resultMetadataMap[resultOperatorFamilies[0].Oid]
			testutils.ExpectStructsToMatchExcluding(&operatorFamily, &resultOperatorFamilies[0], "Oid")
			testutils.ExpectStructsToMatchExcluding(&resultMetadata, &operatorFamilyMetadata, "Oid")
		})
	})
	Describe("PrintCreateOperatorClassStatements", func() {
		It("creates basic operator class", func() {
			operatorClass := backup.OperatorClass{Oid: 0, Schema: "public", Name: "testclass", FamilySchema: "public", FamilyName: "testclass", IndexMethod: "hash", Type: "integer", Default: false, StorageType: "-", Operators: nil, Functions: nil}
			if connection.Version.Before("5") { // Operator families do not exist prior to GPDB5
				operatorClass.FamilySchema = ""
				operatorClass.FamilyName = ""
			}

			backup.PrintCreateOperatorClassStatements(backupfile, toc, []backup.OperatorClass{operatorClass}, nil)

			testutils.AssertQueryRuns(connection, buffer.String())
			if connection.Version.Before("5") {
				defer testutils.AssertQueryRuns(connection, "DROP OPERATOR CLASS public.testclass USING hash")
			} else {
				defer testutils.AssertQueryRuns(connection, "DROP OPERATOR FAMILY public.testclass USING hash CASCADE")
			}

			resultOperatorClasses := backup.GetOperatorClasses(connection)
			Expect(len(resultOperatorClasses)).To(Equal(1))
			testutils.ExpectStructsToMatchExcluding(&operatorClass, &resultOperatorClasses[0], "Oid")
		})
		It("creates complex operator class", func() {
			testutils.SkipIf4(connection)
			operatorClass := backup.OperatorClass{Oid: 0, Schema: "public", Name: "testclass", FamilySchema: "public", FamilyName: "testfam", IndexMethod: "gist", Type: "integer", Default: true, StorageType: "-", Operators: nil, Functions: nil}
			if connection.Version.Before("5") { // Operator families do not exist prior to GPDB5
				operatorClass.FamilySchema = ""
				operatorClass.FamilyName = ""
			}

			expectedRecheck := false
			if connection.Version.Before("6") {
				expectedRecheck = true
			}
			operatorClass.Operators = []backup.OperatorClassOperator{{ClassOid: 0, StrategyNumber: 1, Operator: "=(integer,integer)", Recheck: expectedRecheck}}
			operatorClass.Functions = []backup.OperatorClassFunction{{ClassOid: 0, SupportNumber: 1, FunctionName: "abs(integer)"}}

			testutils.AssertQueryRuns(connection, "CREATE OPERATOR FAMILY public.testfam USING gist")
			defer testutils.AssertQueryRuns(connection, "DROP OPERATOR FAMILY public.testfam USING gist CASCADE")
			backup.PrintCreateOperatorClassStatements(backupfile, toc, []backup.OperatorClass{operatorClass}, nil)

			testutils.AssertQueryRuns(connection, buffer.String())

			resultOperatorClasses := backup.GetOperatorClasses(connection)
			Expect(len(resultOperatorClasses)).To(Equal(1))
			testutils.ExpectStructsToMatchExcluding(&operatorClass, &resultOperatorClasses[0], "Oid", "Operators.ClassOid", "Functions.ClassOid")
		})
		It("creates basic operator class with a comment and owner", func() {
			operatorClassMetadataMap := testutils.DefaultMetadataMap("OPERATOR CLASS", false, true, true)
			operatorClassMetadata := operatorClassMetadataMap[1]

			operatorClass := backup.OperatorClass{Oid: 1, Schema: "public", Name: "testclass", FamilySchema: "public", FamilyName: "testclass", IndexMethod: "hash", Type: "integer", Default: false, StorageType: "-", Operators: nil, Functions: nil}
			if connection.Version.Before("5") { // Operator families do not exist prior to GPDB5
				operatorClass.FamilySchema = ""
				operatorClass.FamilyName = ""
			}

			backup.PrintCreateOperatorClassStatements(backupfile, toc, []backup.OperatorClass{operatorClass}, operatorClassMetadataMap)

			testutils.AssertQueryRuns(connection, buffer.String())
			if connection.Version.Before("5") {
				defer testutils.AssertQueryRuns(connection, "DROP OPERATOR CLASS public.testclass USING hash")
			} else {
				defer testutils.AssertQueryRuns(connection, "DROP OPERATOR FAMILY public.testclass USING hash CASCADE")
			}

			resultOperatorClasses := backup.GetOperatorClasses(connection)
			Expect(len(resultOperatorClasses)).To(Equal(1))
			testutils.ExpectStructsToMatchExcluding(&operatorClass, &resultOperatorClasses[0], "Oid")

			resultMetadataMap := backup.GetMetadataForObjectType(connection, backup.TYPE_OPERATORCLASS)
			resultMetadata := resultMetadataMap[resultOperatorClasses[0].Oid]
			testutils.ExpectStructsToMatchExcluding(&resultMetadata, &operatorClassMetadata, "Oid")

		})
	})
})
