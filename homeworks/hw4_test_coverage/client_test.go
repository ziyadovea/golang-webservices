package main

import (
	"encoding/json"
	"encoding/xml"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

// В структуру выносим только нужные в работе поля
type Row struct {
	XMLName   xml.Name `xml:"row"`
	Id        int      `xml:"id"`
	Age       int      `xml:"age"`
	FirstName string   `xml:"first_name"`
	LastName  string   `xml:"last_name"`
	Gender    string   `xml:"gender"`
	About     string   `xml:"about"`
}

type Root struct {
	XMLName xml.Name `xml:"root"`
	Rows    []Row    `xml:"row"`
}

// Зададим для примера токен
var AccessToken = "ExampleAccessToken"

// Вспомогательные переменные для тестирование
var datasetName = "dataset.xml"
var invalidJsonErrorResponse = false
var invalidUsersJson = false
var sendUnknownBadRequestError = false

func SearchServer(w http.ResponseWriter, r *http.Request) {

	// Проверяем токен, по которому происходит авторизация в системе
	if r.Header.Get("AccessToken") != AccessToken {
		http.Error(w, "incorrect access token", http.StatusUnauthorized)
		return
	}

	// Распакуем dataset.xml в структуру
	xmlFile, err := os.Open(datasetName)
	if err != nil {
		http.Error(w, "error with open source file with users: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer xmlFile.Close()

	byteData, err := ioutil.ReadAll(xmlFile)
	if err != nil {
		http.Error(w, "error with read source file with users: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var root Root
	err = xml.Unmarshal(byteData, &root)
	if err != nil {
		http.Error(w, "error with unmarshal source file with users: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var users []User
	// Обработка запроса и поиск пользователей
	params := r.URL.Query()
	query := params.Get("query")
	for _, row := range root.Rows {
		// Если query не пустой
		if query != "" {
			// Параметр query ищет по полям Name и About
			if !(strings.Contains(row.FirstName+" "+row.LastName, query) ||
				strings.Contains(row.About, query)) {
				continue
			}
		}
		// Если query пустой, то делаем только сортировку, т.е. возвращаем все записи
		users = append(users, User{row.Id,
			row.FirstName + " " + row.LastName,
			row.Age,
			row.About,
			row.Gender,
		})
	}

	// Сортировка
	limit, _ := strconv.Atoi(params.Get("limit"))       // Количество записей
	offset, _ := strconv.Atoi(params.Get("offset"))     // Сдвиг
	order_field := params.Get("order_field")            // Поле для сортировки
	order_by, _ := strconv.Atoi(params.Get("order_by")) // Порядок сортировки

	// Если необходимо отсортировать
	if order_by != OrderByAsIs {
		switch order_field {
		case "Id":
			sort.Slice(users, func(i, j int) bool {
				if order_by == OrderByAsc {
					return users[i].Id < users[j].Id
				}
				return users[i].Id > users[j].Id // OrderByDesc
			})
		case "Age":
			sort.Slice(users, func(i, j int) bool {
				if order_by == OrderByAsc {
					return users[i].Age < users[j].Age
				}
				return users[i].Age > users[j].Age // OrderByDesc
			})
		case "Name", "":
			sort.Slice(users, func(i, j int) bool {
				if order_by == OrderByAsc {
					return users[i].Name < users[j].Name
				}
				return users[i].Name > users[j].Name // OrderByDesc
			})
		default:
			// Здесь надо ругаться не просто http.Error, а отправиить как json
			errResp := SearchErrorResponse{"ErrorBadOrderField"}

			// Для тестирования отправки неизвестной ошибки запроса
			if sendUnknownBadRequestError {
				errResp = SearchErrorResponse{"esskeetit"}
			}

			byteErrResp, err := json.Marshal(errResp)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// Для тестирование того, что отправляется некорретный json
			if invalidJsonErrorResponse {
				byteErrResp = []byte{1, 2, 3, 4, 5, 6, 7, 8}
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write(byteErrResp)
			return
		}
	}

	// Учет limit и offset
	if offset > len(users) {
		users = make([]User, 0)
	} else {
		if limit+offset < len(users) {
			users = users[offset : limit+offset]
		} else {
			users = users[offset:len(users)]
		}
	}

	jsonUsers, err := json.Marshal(users)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Для тестирование того, что отправляется некорретный json
	if invalidUsersJson {
		jsonUsers = []byte{1, 2, 3, 4, 5, 6, 7, 8}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonUsers)

}

// Тестирование FindUsers

func TestLimitLessThanZero(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()
	client := SearchClient{AccessToken, ts.URL}

	_, err := client.FindUsers(SearchRequest{Limit: -1})

	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if err != nil && err.Error() != "limit must be > 0" {
		t.Errorf("unexpected error: %#v", err)
	}
}

func TestLimitMoreThanTwentyFive(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()
	client := SearchClient{AccessToken, ts.URL}

	req, err := client.FindUsers(SearchRequest{Limit: 26})

	if err != nil {
		t.Errorf("unexpected error: %#v", err)
	}
	if len(req.Users) != 25 {
		t.Errorf("incorrect numer of users: %d", len(req.Users))
	}
}

func TestOffsetLessThanZero(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()
	client := SearchClient{AccessToken, ts.URL}

	_, err := client.FindUsers(SearchRequest{Offset: -1})

	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if err != nil && err.Error() != "offset must be > 0" {
		t.Errorf("unexpected error: %#v", err)
	}
}

func TestBadAccessToken(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()
	client := SearchClient{"Bad" + AccessToken, ts.URL}

	_, err := client.FindUsers(SearchRequest{})

	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if err != nil && err.Error() != "Bad AccessToken" {
		t.Errorf("unexpected error: %#v", err)
	}
}

func TestDatasetName(t *testing.T) {

	datasetName = "error"

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()
	client := SearchClient{AccessToken, ts.URL}

	_, err := client.FindUsers(SearchRequest{})

	datasetName = "dataset.xml"

	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if err != nil && err.Error() != "SearchServer fatal error" {
		t.Errorf("unexpected error: %#v", err)
	}
}

func TestInvalidOrderField(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()
	client := SearchClient{AccessToken, ts.URL}

	_, err := client.FindUsers(SearchRequest{OrderField: "InvalidOrderField", OrderBy: OrderByDesc})

	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if err != nil && err.Error() != "OrderFeld InvalidOrderField invalid" {
		t.Errorf("unexpected error: %#v", err)
	}
}

func TestInvalidJsonErrorResponse(t *testing.T) {

	invalidJsonErrorResponse = true

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()
	client := SearchClient{AccessToken, ts.URL}

	_, err := client.FindUsers(SearchRequest{OrderField: "InvalidOrderField", OrderBy: OrderByDesc})

	invalidJsonErrorResponse = false

	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "cant unpack error json") {
		t.Errorf("unexpected error: %#v", err)
	}

}

func TestUnknownBadRequestError(t *testing.T) {

	sendUnknownBadRequestError = true

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()
	client := SearchClient{AccessToken, ts.URL}

	_, err := client.FindUsers(SearchRequest{OrderField: "InvalidOrderField", OrderBy: OrderByDesc})

	sendUnknownBadRequestError = false

	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "unknown bad request error") {
		t.Errorf("unexpected error: %#v", err)
	}

}

func TestInvalidUsersJson(t *testing.T) {

	invalidUsersJson = true

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()
	client := SearchClient{AccessToken, ts.URL}

	_, err := client.FindUsers(SearchRequest{OrderField: "Name", OrderBy: OrderByDesc})

	invalidUsersJson = false

	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "cant unpack result json") {
		t.Errorf("unexpected error: %#v", err)
	}

}

func TestClientDoUnknownError(t *testing.T) {
	client := SearchClient{AccessToken, ""}
	_, err := client.FindUsers(SearchRequest{OrderField: "Name", OrderBy: OrderByDesc})
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "unknown error") {
		t.Errorf("unexpected error: %#v", err)
	}
}

func TestClientDoTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
	}))
	defer ts.Close()
	client := SearchClient{AccessToken, ts.URL}
	_, err := client.FindUsers(SearchRequest{OrderField: "Name", OrderBy: OrderByDesc})
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "timeout for") {
		t.Errorf("unexpected error: %#v", err)
	}
}

func TestLenClient(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()
	client := SearchClient{
		AccessToken: AccessToken,
		URL:         ts.URL,
	}
	_, err := client.FindUsers(SearchRequest{
		Limit:      100,
		Offset:     0,
		Query:      "Annie",
		OrderField: "",
		OrderBy:    0,
	})
	if err != nil {
		t.Errorf("unexpected error: %#v", err)
	}
}
