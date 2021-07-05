package generator

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/zhaoshuyi-s0221/go-queryset/internal/parser"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/zhaoshuyi-s0221/go-queryset/internal/queryset/generator/test"
	assert "github.com/stretchr/testify/require"

	sqlmock "gopkg.in/DATA-DOG/go-sqlmock.v1"
)

const testSurname = "Ivanov"

func fixedFullRe(s string) string {
	return fmt.Sprintf("^%s$", regexp.QuoteMeta(s))
}

func newDB() (sqlmock.Sqlmock, *gorm.DB) {
	db, mock, err := sqlmock.New()
	if err != nil {
		log.Fatalf("can't create sqlmock: %s", err)
	}

	gormDB, gerr := gorm.Open("mysql", db)
	if gerr != nil {
		log.Fatalf("can't open gorm connection: %s", err)
	}
	gormDB.LogMode(true)

	return mock, gormDB.Set("gorm:update_column", true)
}

func getRowsForUsers(users []test.User) *sqlmock.Rows {
	var userFieldNames = []string{"id", "name", "user_surname", "email", "created_at", "updated_at", "deleted_at"}
	rows := sqlmock.NewRows(userFieldNames)

	for _, u := range users {
		rows = rows.AddRow(u.ID, u.Name, u.Surname, u.Email, u.CreatedAt, u.UpdatedAt, u.DeletedAt)
	}
	return rows
}

func getRowWithFields(fields []driver.Value) *sqlmock.Rows {
	fieldNames := []string{}
	for i := range fields {
		fieldNames = append(fieldNames, fmt.Sprintf("f%d", i))
	}

	return sqlmock.NewRows(fieldNames).AddRow(fields...)
}

func getTestUsers(n int) []test.User {
	ret := []test.User{}

	for i := 0; i < n; i++ {
		u := test.User{
			Model: gorm.Model{
				ID:        uint(i),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			Email: fmt.Sprintf("u%d@mail.ru", i),
			Name:  fmt.Sprintf("name_%d", i),
		}
		ret = append(ret, u)
	}

	return ret
}

func getUserNoID() test.User {
	randInt := rand.Int()
	return test.User{
		Model: gorm.Model{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Email: fmt.Sprintf("qs_%d@mail.ru", randInt),
		Name:  fmt.Sprintf("name_rand_%d", randInt),
	}
}

func getUser() test.User {
	u := getUserNoID()
	u.ID = uint(rand.Int())
	return u
}

func checkMock(t *testing.T, mock sqlmock.Sqlmock) {
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expections: %s", err)
	}
}

type testQueryFunc func(t *testing.T, m sqlmock.Sqlmock, db *gorm.DB)

func TestQueries(t *testing.T) {
	funcs := []testQueryFunc{
		testUserSelectAll,
		testUserSelectAllSingleField,
		testUserSelectAllMultipleFields,
		testUserSelectWithLimitAndOffset,
		testUserSelectAllNoRecords,
		testUserSelectOne,
		testUserSelectWithSurnameFilter,
		testUserCreateOne,
		testUserCreateOneWithSurname,
		testUserUpdateFieldsByPK,
		testUserUpdateByEmail,
		testUserDeleteByEmail,
		testUserDeleteByPK,
		testUserQueryFilters,
		testUsersCount,
		testUsersUpdateNum,
		testUsersDeleteNum,
		testUsersDeleteNumUnscoped,
	}
	for _, f := range funcs {
		f := f // save range var
		funcName := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
		funcName = filepath.Ext(funcName)
		funcName = strings.TrimPrefix(funcName, ".")
		t.Run(funcName, func(t *testing.T) {
			t.Parallel()
			m, db := newDB()
			defer checkMock(t, m)
			f(t, m, db)
		})
	}
}

func testUserSelectAll(t *testing.T, m sqlmock.Sqlmock, db *gorm.DB) {
	expUsers := getTestUsers(2)
	m.ExpectQuery(fixedFullRe("SELECT * FROM `users` WHERE `users`.`deleted_at` IS NULL")).
		WillReturnRows(getRowsForUsers(expUsers))

	var users []test.User

	assert.Nil(t, test.NewUserQuerySet(db).All(&users))
	assert.Equal(t, expUsers, users)
}

func testUserSelectAllSingleField(t *testing.T, m sqlmock.Sqlmock, db *gorm.DB) {
	expUsers := getTestUsers(2)
	m.ExpectQuery(fixedFullRe("SELECT name FROM `users` WHERE `users`.`deleted_at` IS NULL")).
		WillReturnRows(getRowsForUsers(expUsers))

	var users []test.User

	assert.Nil(t, test.NewUserQuerySet(db).Select(test.UserDBSchema.Name).All(&users))
	assert.Equal(t, expUsers, users)
}

func testUserSelectAllMultipleFields(t *testing.T, m sqlmock.Sqlmock, db *gorm.DB) {
	expUsers := getTestUsers(2)
	m.ExpectQuery(fixedFullRe("SELECT name,email FROM `users` WHERE `users`.`deleted_at` IS NULL")).
		WillReturnRows(getRowsForUsers(expUsers))

	var users []test.User

	assert.Nil(t, test.NewUserQuerySet(db).Select(test.UserDBSchema.Name, test.UserDBSchema.Email).All(&users))
	assert.Equal(t, expUsers, users)
}

func testUserSelectWithLimitAndOffset(t *testing.T, m sqlmock.Sqlmock, db *gorm.DB) {
	expUsers := getTestUsers(2)
	req := "SELECT * FROM `users` WHERE `users`.`deleted_at` IS NULL LIMIT 1 OFFSET 1"
	m.ExpectQuery(fixedFullRe(req)).
		WillReturnRows(getRowsForUsers(expUsers))

	var users []test.User

	assert.Nil(t, test.NewUserQuerySet(db).Limit(1).Offset(1).All(&users))
	assert.Equal(t, expUsers[0], users[0])
}

func testUserSelectAllNoRecords(t *testing.T, m sqlmock.Sqlmock, db *gorm.DB) {
	m.ExpectQuery(fixedFullRe("SELECT * FROM `users` WHERE `users`.`deleted_at` IS NULL")).
		WillReturnError(sql.ErrNoRows)

	var users []test.User

	assert.Error(t, gorm.ErrRecordNotFound, test.NewUserQuerySet(db).All(&users))
	assert.Len(t, users, 0)
}

func testUserSelectOne(t *testing.T, m sqlmock.Sqlmock, db *gorm.DB) {
	expUsers := getTestUsers(1)
	req := "SELECT * FROM `users` WHERE `users`.`deleted_at` IS NULL ORDER BY `users`.`id` ASC LIMIT 1"
	m.ExpectQuery(fixedFullRe(req)).
		WillReturnRows(getRowsForUsers(expUsers))

	var user test.User

	assert.Nil(t, test.NewUserQuerySet(db).One(&user))
	assert.Equal(t, expUsers[0], user)
}

func testUserSelectWithSurnameFilter(t *testing.T, m sqlmock.Sqlmock, db *gorm.DB) {
	expUsers := getTestUsers(1)

	surname := testSurname
	expUsers[0].Surname = &surname

	req := "SELECT * FROM `users` " +
		"WHERE `users`.`deleted_at` IS NULL AND ((user_surname = ?)) ORDER BY `users`.`id` ASC LIMIT 1"
	m.ExpectQuery(fixedFullRe(req)).
		WillReturnRows(getRowsForUsers(expUsers))

	var user test.User

	assert.Nil(t, test.NewUserQuerySet(db).SurnameEq(surname).One(&user))
	assert.Equal(t, expUsers[0], user)
}

type qsQuerier func(qs test.UserQuerySet) test.UserQuerySet

type userQueryTestCase struct {
	q    string
	args []driver.Value
	qs   qsQuerier
}

func testUserQueryFilters(t *testing.T, m sqlmock.Sqlmock, db *gorm.DB) {
	cases := []userQueryTestCase{
		{
			q:    "((name IN (?)))",
			args: []driver.Value{"a"},
			qs: func(qs test.UserQuerySet) test.UserQuerySet {
				return qs.NameIn("a")
			},
		},
		{
			q:    "((name IN (?,?)))",
			args: []driver.Value{"a", "b"},
			qs: func(qs test.UserQuerySet) test.UserQuerySet {
				return qs.NameIn("a", "b")
			},
		},
		{
			q:    "((name NOT IN (?)))",
			args: []driver.Value{"a"},
			qs: func(qs test.UserQuerySet) test.UserQuerySet {
				return qs.NameNotIn("a")
			},
		},
		{
			q:    "((name NOT IN (?,?)))",
			args: []driver.Value{"a", "b"},
			qs: func(qs test.UserQuerySet) test.UserQuerySet {
				return qs.NameNotIn("a", "b")
			},
		},
	}
	for _, c := range cases {
		t.Run(c.q, func(t *testing.T) {
			runUserQueryFilterSubTest(t, c, m, db)
		})
	}
}

func runUserQueryFilterSubTest(t *testing.T, c userQueryTestCase, m sqlmock.Sqlmock, db *gorm.DB) {
	expUsers := getTestUsers(5)
	req := "SELECT * FROM `users` WHERE `users`.`deleted_at` IS NULL AND " + c.q
	m.ExpectQuery(fixedFullRe(req)).WithArgs(c.args...).
		WillReturnRows(getRowsForUsers(expUsers))

	var users []test.User

	assert.Nil(t, c.qs(test.NewUserQuerySet(db)).All(&users))
	assert.Equal(t, expUsers, users)
}

func testUserCreateOne(t *testing.T, m sqlmock.Sqlmock, db *gorm.DB) {
	u := getUserNoID()
	req := "INSERT INTO `users` (`created_at`,`updated_at`,`deleted_at`,`name`,`user_surname`,`email`) " +
		"VALUES (?,?,?,?,?,?)"

	args := []driver.Value{sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
		u.Name, nil, u.Email}
	m.ExpectExec(fixedFullRe(req)).
		WithArgs(args...).
		WillReturnResult(sqlmock.NewResult(2, 1))
	assert.Nil(t, u.Create(db))
	assert.Equal(t, uint(2), u.ID)
}

func testUserCreateOneWithSurname(t *testing.T, m sqlmock.Sqlmock, db *gorm.DB) {
	u := getUserNoID()
	req := "INSERT INTO `users` (`created_at`,`updated_at`,`deleted_at`,`name`,`user_surname`,`email`) " +
		"VALUES (?,?,?,?,?,?)"

	surname := testSurname
	u.Surname = &surname

	args := []driver.Value{sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
		u.Name, &surname, u.Email}
	m.ExpectExec(fixedFullRe(req)).
		WithArgs(args...).
		WillReturnResult(sqlmock.NewResult(2, 1))
	assert.Nil(t, u.Create(db))
	assert.Equal(t, uint(2), u.ID)
}

func testUserUpdateByEmail(t *testing.T, m sqlmock.Sqlmock, db *gorm.DB) {
	u := getUser()
	req := "UPDATE `users` SET `name` = ? WHERE `users`.`deleted_at` IS NULL AND ((email = ?))"
	m.ExpectExec(fixedFullRe(req)).
		WithArgs(u.Name, u.Email).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := test.NewUserQuerySet(db).
		EmailEq(u.Email).
		GetUpdater().
		SetName(u.Name).
		Update()
	assert.Nil(t, err)
}

func testUserUpdateFieldsByPK(t *testing.T, m sqlmock.Sqlmock, db *gorm.DB) {
	u := getUser()
	req := "UPDATE `users` SET `name` = ? WHERE `users`.`deleted_at` IS NULL AND `users`.`id` = ?"
	m.ExpectExec(fixedFullRe(req)).
		WithArgs(u.Name, u.ID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	assert.Nil(t, u.Update(db, test.UserDBSchema.Name))
}

func testUserDeleteByEmail(t *testing.T, m sqlmock.Sqlmock, db *gorm.DB) {
	u := getUser()
	req := "UPDATE `users` SET `deleted_at`=? WHERE `users`.`deleted_at` IS NULL AND ((email = ?))"
	m.ExpectExec(fixedFullRe(req)).
		WithArgs(sqlmock.AnyArg(), u.Email).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := test.NewUserQuerySet(db).
		EmailEq(u.Email).
		Delete()
	assert.Nil(t, err)
}

func testUserDeleteByPK(t *testing.T, m sqlmock.Sqlmock, db *gorm.DB) {
	u := getUser()
	req := "UPDATE `users` SET `deleted_at`=? WHERE `users`.`deleted_at` IS NULL AND `users`.`id` = ?"
	m.ExpectExec(fixedFullRe(req)).
		WithArgs(sqlmock.AnyArg(), u.ID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	assert.Nil(t, u.Delete(db))
}

func testUsersDeleteNum(t *testing.T, m sqlmock.Sqlmock, db *gorm.DB) {
	usersNum := 2
	users := getTestUsers(usersNum)
	req := "UPDATE `users` SET `deleted_at`=? WHERE `users`.`deleted_at` IS NULL AND ((email IN (?,?)))"
	m.ExpectExec(fixedFullRe(req)).
		WithArgs(sqlmock.AnyArg(), users[0].Email, users[1].Email).
		WillReturnResult(sqlmock.NewResult(0, int64(usersNum)))

	num, err := test.NewUserQuerySet(db).
		EmailIn(users[0].Email, users[1].Email).
		DeleteNum()
	assert.Nil(t, err)
	assert.Equal(t, int64(usersNum), num)
}

func testUsersDeleteNumUnscoped(t *testing.T, m sqlmock.Sqlmock, db *gorm.DB) {
	usersNum := 2
	users := getTestUsers(usersNum)
	req := "DELETE FROM `users` WHERE (email IN (?,?))"
	m.ExpectExec(fixedFullRe(req)).
		WithArgs(users[0].Email, users[1].Email).
		WillReturnResult(sqlmock.NewResult(0, int64(usersNum)))

	num, err := test.NewUserQuerySet(db).
		EmailIn(users[0].Email, users[1].Email).
		DeleteNumUnscoped()
	assert.Nil(t, err)
	assert.Equal(t, int64(usersNum), num)
}

func testUsersUpdateNum(t *testing.T, m sqlmock.Sqlmock, db *gorm.DB) {
	usersNum := 2
	users := getTestUsers(usersNum)
	req := "UPDATE `users` SET `name` = ? WHERE `users`.`deleted_at` IS NULL AND ((email IN (?,?)))"
	m.ExpectExec(fixedFullRe(req)).
		WithArgs(sqlmock.AnyArg(), users[0].Email, users[1].Email).
		WillReturnResult(sqlmock.NewResult(0, int64(usersNum)))

	num, err := test.NewUserQuerySet(db).
		EmailIn(users[0].Email, users[1].Email).
		GetUpdater().
		SetName("some name").
		UpdateNum()
	assert.Nil(t, err)
	assert.Equal(t, int64(usersNum), num)
}

func testUsersCount(t *testing.T, m sqlmock.Sqlmock, db *gorm.DB) {
	expCount := 5
	req := "SELECT count(*) FROM `users` WHERE `users`.`deleted_at` IS NULL AND ((name != ?))"
	m.ExpectQuery(fixedFullRe(req)).WithArgs(driver.Value("")).
		WillReturnRows(getRowWithFields([]driver.Value{expCount}))

	cnt, err := test.NewUserQuerySet(db).NameNe("").Count()
	assert.Nil(t, err)
	assert.Equal(t, expCount, cnt)
}

func TestMain(m *testing.M) {
	g := Generator{
		StructsParser: &parser.Structs{},
	}
	err := g.Generate(context.Background(), "test/models.go", "test/autogenerated_models.go")
	if err != nil {
		panic(err)
	}

	os.Exit(m.Run())
}

func BenchmarkHello(b *testing.B) {
	g := Generator{
		StructsParser: &parser.Structs{},
	}

	for i := 0; i < b.N; i++ {
		err := g.Generate(context.Background(), "test/models.go", "test/autogenerated_models.go")
		if err != nil {
			b.Fatalf("can't generate querysets: %s", err)
		}
	}
}
