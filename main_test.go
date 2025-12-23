package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCafeNegative(t *testing.T) {
	handler := http.HandlerFunc(mainHandle)

	requests := []struct {
		request string
		status  int
		message string
	}{
		{"/cafe", http.StatusBadRequest, "unknown city"},
		{"/cafe?city=omsk", http.StatusBadRequest, "unknown city"},
		{"/cafe?city=tula&count=na", http.StatusBadRequest, "incorrect count"},
	}
	for _, v := range requests {
		response := httptest.NewRecorder()
		req := httptest.NewRequest("GET", v.request, nil)
		handler.ServeHTTP(response, req)

		assert.Equal(t, v.status, response.Code)
		assert.Equal(t, v.message, strings.TrimSpace(response.Body.String()))
	}
}

func TestCafeWhenOk(t *testing.T) {
	handler := http.HandlerFunc(mainHandle)

	requests := []string{
		"/cafe?count=2&city=moscow",
		"/cafe?city=tula",
		"/cafe?city=moscow&search=ложка",
	}
	for _, v := range requests {
		response := httptest.NewRecorder()
		req := httptest.NewRequest("GET", v, nil)

		handler.ServeHTTP(response, req)

		assert.Equal(t, http.StatusOK, response.Code)
	}
}

func TestCafeCount(t *testing.T) {
	requests := []struct {
		count int
		want  int
	}{
		{count: 0, want: 0},   // Пустая строка (""), так как count=0
		{count: 1, want: 1},   // Одно кафе
		{count: 2, want: 2},   // Два кафе
		{count: 100, want: 5}, // Максимум кафе в Москве
	}

	ts := httptest.NewServer(http.HandlerFunc(mainHandle))
	defer ts.Close()

	for _, req := range requests {
		t.Run(fmt.Sprintf("count=%d", req.count), func(t *testing.T) {
			url := ts.URL + "/cafe?city=moscow&count=" + strconv.Itoa(req.count)
			resp, err := http.Get(url)
			require.NoError(t, err, "HTTP request failed")
			require.Equal(t, http.StatusOK, resp.StatusCode, "Expected status code 200")
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err, "Failed to read response body")

			cafes := strings.Split(string(body), ",")
			if len(cafes) == 1 && cafes[0] == "" {
				cafes = []string{} // Исправление для пустого результата
			}
			assert.Len(t, cafes, req.want, "Unexpected number of cafes for count=%d", req.count)
		})
	}
}

func TestCafeSearch(t *testing.T) {
	requests := []struct {
		search    string
		wantCount int
	}{
		{search: "фасоль", wantCount: 0}, // Ничего не найдено
		{search: "кофе", wantCount: 2},   // Два кафе
		{search: "вилка", wantCount: 1},  // Одно кафе
	}

	ts := httptest.NewServer(http.HandlerFunc(mainHandle))
	defer ts.Close()

	for _, req := range requests {
		t.Run(fmt.Sprintf("search=%s", req.search), func(t *testing.T) {
			url := ts.URL + "/cafe?city=moscow&search=" + req.search
			resp, err := http.Get(url)
			require.NoError(t, err, "HTTP request failed")
			require.Equal(t, http.StatusOK, resp.StatusCode, "Expected status code 200")
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err, "Failed to read response body")

			cafes := strings.Split(string(body), ",")
			if len(cafes) == 1 && cafes[0] == "" {
				cafes = []string{} // Исправление для пустого результата
			}
			assert.Len(t, cafes, req.wantCount, "Unexpected number of cafes for search=%s", req.search)

			searchLower := strings.ToLower(req.search)
			for _, cafe := range cafes {
				if cafe != "" {
					assert.Contains(t, strings.ToLower(cafe), searchLower, "Cafe '%s' does not contain search string '%s'", cafe, req.search)
				}
			}
		})
	}
}
