package integration

import (
	"regexp"

	"github.com/greenplum-db/gpbackup/backup"
	"github.com/greenplum-db/gpbackup/testutils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("backup integration create statement tests", func() {
	BeforeEach(func() {
		toc, backupfile = testutils.InitializeTestTOC(buffer, "predata")
	})
	Describe("PrintCreateResourceQueueStatements", func() {
		It("creates a basic resource queue with a comment", func() {
			basicQueue := backup.ResourceQueue{Oid: 1, Name: `"basicQueue"`, ActiveStatements: -1, MaxCost: "32.80", CostOvercommit: false, MinCost: "0.00", Priority: "medium", MemoryLimit: "-1"}
			resQueueMetadataMap := testutils.DefaultMetadataMap("RESOURCE QUEUE", false, false, true)
			resQueueMetadata := resQueueMetadataMap[1]

			backup.PrintCreateResourceQueueStatements(backupfile, toc, []backup.ResourceQueue{basicQueue}, resQueueMetadataMap)

			// CREATE RESOURCE QUEUE statements can not be part of a multi-command statement, so
			// feed the CREATE RESOURCE QUEUE and COMMENT ON statements separately.
			hunks := regexp.MustCompile(";\n\n").Split(buffer.String(), 2)
			testutils.AssertQueryRuns(connection, hunks[0])
			defer testutils.AssertQueryRuns(connection, `DROP RESOURCE QUEUE "basicQueue"`)
			testutils.AssertQueryRuns(connection, hunks[1])

			resultResourceQueues := backup.GetResourceQueues(connection)
			resQueueOid := testutils.OidFromObjectName(connection, "", "basicQueue", backup.TYPE_RESOURCEQUEUE)
			resultMetadataMap := backup.GetCommentsForObjectType(connection, backup.TYPE_RESOURCEQUEUE)
			resultMetadata := resultMetadataMap[resQueueOid]
			testutils.ExpectStructsToMatch(&resultMetadata, &resQueueMetadata)

			for _, resultQueue := range resultResourceQueues {
				if resultQueue.Name == `"basicQueue"` {
					testutils.ExpectStructsToMatchExcluding(&basicQueue, &resultQueue, "Oid")
					return
				}
			}
		})
		It("creates a resource queue with all attributes", func() {
			everythingQueue := backup.ResourceQueue{Oid: 1, Name: `"everythingQueue"`, ActiveStatements: 7, MaxCost: "32.80", CostOvercommit: true, MinCost: "22.80", Priority: "low", MemoryLimit: "2GB"}
			emptyMetadataMap := map[uint32]backup.ObjectMetadata{}

			backup.PrintCreateResourceQueueStatements(backupfile, toc, []backup.ResourceQueue{everythingQueue}, emptyMetadataMap)

			testutils.AssertQueryRuns(connection, buffer.String())
			defer testutils.AssertQueryRuns(connection, `DROP RESOURCE QUEUE "everythingQueue"`)

			resultResourceQueues := backup.GetResourceQueues(connection)

			for _, resultQueue := range resultResourceQueues {
				if resultQueue.Name == `"everythingQueue"` {
					testutils.ExpectStructsToMatchExcluding(&everythingQueue, &resultQueue, "Oid")
					return
				}
			}
			Fail("Could not find everythingQueue")
		})
	})
	Describe("PrintCreateResourceGroupStatements", func() {
		BeforeEach(func() {
			testutils.SkipIf4(connection)
		})
		It("creates a basic resource group", func() {
			someGroup := backup.ResourceGroup{Oid: 1, Name: "some_group", CPURateLimit: 10, MemoryLimit: 20, Concurrency: 15, MemorySharedQuota: 25, MemorySpillRatio: 30}
			emptyMetadataMap := map[uint32]backup.ObjectMetadata{}

			backup.PrintCreateResourceGroupStatements(backupfile, toc, []backup.ResourceGroup{someGroup}, emptyMetadataMap)

			testutils.AssertQueryRuns(connection, buffer.String())
			defer testutils.AssertQueryRuns(connection, `DROP RESOURCE GROUP some_group`)

			resultResourceGroups := backup.GetResourceGroups(connection)

			for _, resultGroup := range resultResourceGroups {
				if resultGroup.Name == "some_group" {
					testutils.ExpectStructsToMatchExcluding(&someGroup, &resultGroup, "Oid")
					return
				}
			}
			Fail("Could not find some_group")
		})
		It("alters a default resource group", func() {
			defaultGroup := backup.ResourceGroup{Oid: 1, Name: "default_group", CPURateLimit: 10, MemoryLimit: 20, Concurrency: 15, MemorySharedQuota: 25, MemorySpillRatio: 30}
			emptyMetadataMap := map[uint32]backup.ObjectMetadata{}

			backup.PrintCreateResourceGroupStatements(backupfile, toc, []backup.ResourceGroup{defaultGroup}, emptyMetadataMap)

			hunks := regexp.MustCompile(";\n\n").Split(buffer.String(), 5)
			for i := 0; i < 5; i++ {
				testutils.AssertQueryRuns(connection, hunks[i])
			}
			resultResourceGroups := backup.GetResourceGroups(connection)

			for _, resultGroup := range resultResourceGroups {
				if resultGroup.Name == "default_group" {
					testutils.ExpectStructsToMatchExcluding(&defaultGroup, &resultGroup, "Oid")
					return
				}
			}
			Fail("Could not find default_group")
		})
	})
	Describe("PrintCreateRoleStatements", func() {
		It("creates a basic role ", func() {
			role1 := backup.Role{
				Oid:             0,
				Name:            "role1",
				Super:           true,
				Inherit:         false,
				CreateRole:      false,
				CreateDB:        false,
				CanLogin:        false,
				ConnectionLimit: -1,
				Password:        "",
				ValidUntil:      "",
				ResQueue:        "pg_default",
				ResGroup:        "default_group",
				Createrexthttp:  false,
				Createrextgpfd:  false,
				Createwextgpfd:  false,
				Createrexthdfs:  false,
				Createwexthdfs:  false,
				TimeConstraints: nil,
			}
			if connection.Version.Before("5") {
				role1.ResGroup = ""
			}
			emptyMetadataMap := backup.MetadataMap{}

			backup.PrintCreateRoleStatements(backupfile, toc, []backup.Role{role1}, emptyMetadataMap)

			testutils.AssertQueryRuns(connection, buffer.String())
			defer testutils.AssertQueryRuns(connection, `DROP ROLE "role1"`)
			role1.Oid = testutils.OidFromObjectName(connection, "", "role1", backup.TYPE_ROLE)

			resultRoles := backup.GetRoles(connection)
			for _, role := range resultRoles {
				if role.Name == "role1" {
					testutils.ExpectStructsToMatch(&role1, role)
					return
				}
			}
			Fail("Role 'role1' was not found")
		})
		It("creates a role with all attributes", func() {
			role1 := backup.Role{
				Oid:             1,
				Name:            "role1",
				Super:           false,
				Inherit:         true,
				CreateRole:      true,
				CreateDB:        true,
				CanLogin:        true,
				ConnectionLimit: 4,
				Password:        "md5a8b2c77dfeba4705f29c094592eb3369",
				ValidUntil:      "2099-01-01 08:00:00-00",
				ResQueue:        "pg_default",
				ResGroup:        "default_group",
				Createrexthttp:  true,
				Createrextgpfd:  true,
				Createwextgpfd:  true,
				Createrexthdfs:  true,
				Createwexthdfs:  true,
				TimeConstraints: []backup.TimeConstraint{
					{
						Oid:       0,
						StartDay:  0,
						StartTime: "13:30:00",
						EndDay:    3,
						EndTime:   "14:30:00",
					}, {
						Oid:       0,
						StartDay:  5,
						StartTime: "00:00:00",
						EndDay:    5,
						EndTime:   "24:00:00",
					},
				},
			}
			if connection.Version.Before("5") {
				role1.ResGroup = ""
			}
			metadataMap := testutils.DefaultMetadataMap("ROLE", false, false, true)

			backup.PrintCreateRoleStatements(backupfile, toc, []backup.Role{role1}, metadataMap)

			testutils.AssertQueryRuns(connection, buffer.String())
			defer testutils.AssertQueryRuns(connection, `DROP ROLE "role1"`)
			role1.Oid = testutils.OidFromObjectName(connection, "", "role1", backup.TYPE_ROLE)

			resultRoles := backup.GetRoles(connection)
			for _, role := range resultRoles {
				if role.Name == "role1" {
					testutils.ExpectStructsToMatchExcluding(&role1, role, "TimeConstraints.Oid")
					return
				}
			}
			Fail("Role 'role1' was not found")
		})
	})
	Describe("PrintRoleMembershipStatements", func() {
		BeforeEach(func() {
			testutils.AssertQueryRuns(connection, `CREATE ROLE usergroup`)
			testutils.AssertQueryRuns(connection, `CREATE ROLE testuser`)
		})
		AfterEach(func() {
			defer testutils.AssertQueryRuns(connection, `DROP ROLE usergroup`)
			defer testutils.AssertQueryRuns(connection, `DROP ROLE testuser`)
		})
		It("grants a role without ADMIN OPTION", func() {
			numRoleMembers := len(backup.GetRoleMembers(connection))
			expectedRoleMember := backup.RoleMember{Role: "usergroup", Member: "testuser", Grantor: "testrole", IsAdmin: false}
			backup.PrintRoleMembershipStatements(backupfile, toc, []backup.RoleMember{expectedRoleMember})

			testutils.AssertQueryRuns(connection, buffer.String())

			resultRoleMembers := backup.GetRoleMembers(connection)
			Expect(len(resultRoleMembers)).To(Equal(numRoleMembers + 1))
			for _, roleMember := range resultRoleMembers {
				if roleMember.Role == "usergroup" {
					testutils.ExpectStructsToMatch(&expectedRoleMember, &roleMember)
					return
				}
			}
			Fail("Role 'testuser' is not a member of role 'usergroup'")
		})
		It("grants a role WITH ADMIN OPTION", func() {
			numRoleMembers := len(backup.GetRoleMembers(connection))
			expectedRoleMember := backup.RoleMember{Role: "usergroup", Member: "testuser", Grantor: "testrole", IsAdmin: true}
			backup.PrintRoleMembershipStatements(backupfile, toc, []backup.RoleMember{expectedRoleMember})

			testutils.AssertQueryRuns(connection, buffer.String())

			resultRoleMembers := backup.GetRoleMembers(connection)
			Expect(len(resultRoleMembers)).To(Equal(numRoleMembers + 1))
			for _, roleMember := range resultRoleMembers {
				if roleMember.Role == "usergroup" {
					testutils.ExpectStructsToMatch(&expectedRoleMember, &roleMember)
					return
				}
			}
			Fail("Role 'testuser' is not a member of role 'usergroup'")
		})
	})
	Describe("PrintCreateTablespaceStatements", func() {
		expectedTablespace := backup.Tablespace{Oid: 1, Tablespace: "test_tablespace", Filespace: "test_filespace"}
		It("creates a basic tablespace", func() {
			numTablespaces := len(backup.GetTablespaces(connection))
			emptyMetadataMap := backup.MetadataMap{}
			backup.PrintCreateTablespaceStatements(backupfile, toc, []backup.Tablespace{expectedTablespace}, emptyMetadataMap)

			testutils.AssertQueryRuns(connection, buffer.String())
			defer testutils.AssertQueryRuns(connection, "DROP TABLESPACE test_tablespace")

			resultTablespaces := backup.GetTablespaces(connection)
			Expect(len(resultTablespaces)).To(Equal(numTablespaces + 1))
			for _, tablespace := range resultTablespaces {
				if tablespace.Tablespace == "test_tablespace" {
					testutils.ExpectStructsToMatchExcluding(&expectedTablespace, &tablespace, "Oid")
					return
				}
			}
			Fail("Tablespace 'test_tablespace' was not created")
		})
		It("creates a tablespace with permissions, an owner, and a comment", func() {
			numTablespaces := len(backup.GetTablespaces(connection))
			tablespaceMetadataMap := testutils.DefaultMetadataMap("TABLESPACE", true, true, true)
			tablespaceMetadata := tablespaceMetadataMap[1]
			backup.PrintCreateTablespaceStatements(backupfile, toc, []backup.Tablespace{expectedTablespace}, tablespaceMetadataMap)

			testutils.AssertQueryRuns(connection, buffer.String())
			defer testutils.AssertQueryRuns(connection, "DROP TABLESPACE test_tablespace")

			resultTablespaces := backup.GetTablespaces(connection)
			resultMetadataMap := backup.GetMetadataForObjectType(connection, backup.TYPE_TABLESPACE)
			oid := testutils.OidFromObjectName(connection, "", "test_tablespace", backup.TYPE_TABLESPACE)
			resultMetadata := resultMetadataMap[oid]
			testutils.ExpectStructsToMatchExcluding(&tablespaceMetadata, &resultMetadata, "Oid")
			Expect(len(resultTablespaces)).To(Equal(numTablespaces + 1))
			for _, tablespace := range resultTablespaces {
				if tablespace.Tablespace == "test_tablespace" {
					testutils.ExpectStructsToMatchExcluding(&expectedTablespace, &tablespace, "Oid")
					return
				}
			}
			Fail("Tablespace 'test_tablespace' was not created")
		})
	})
})
