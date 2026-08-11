package main

import "testing"

func TestSortFileEntries(t *testing.T) {
	Scenario(t, "GIVEN 폴더와 파일이 섞인 목록 WHEN 정렬 THEN 폴더가 먼저, 그다음 파일이 오고 각각 이름순이다", func(t *testing.T) {
		entries := []fileEntry{
			{name: "b.txt", isDir: false},
			{name: "photos", isDir: true},
			{name: "a.txt", isDir: false},
			{name: "docs", isDir: true},
		}
		sortFileEntries(entries)

		want := []string{"docs", "photos", "a.txt", "b.txt"}
		for i, w := range want {
			if entries[i].name != w {
				t.Fatalf("entries[%d] = %q, 기대값 %q (전체: %+v)", i, entries[i].name, w, entries)
			}
		}
	})

	// 부정/경계 케이스: 빈 디렉터리.
	Scenario(t, "GIVEN 빈 목록 WHEN 정렬 THEN panic 없이 그대로 빈 채로 남는다 (경계 케이스)", func(t *testing.T) {
		var entries []fileEntry
		sortFileEntries(entries) // panic이 없어야 함
		if len(entries) != 0 {
			t.Errorf("빈 목록이 유지돼야 함")
		}
	})
}

func TestFileEntryLabel(t *testing.T) {
	entries := []fileEntry{
		{name: "photos", isDir: true},
		{name: "readme.txt", isDir: false},
	}

	Scenario(t, "GIVEN 인덱스 0 WHEN 라벨 생성 THEN 항상 상위 폴더(..) 항목이다", func(t *testing.T) {
		if got := fileEntryLabel(0, entries); got != "📁 .." {
			t.Errorf("got %q, 기대값 %q", got, "📁 ..")
		}
	})

	Scenario(t, "GIVEN 폴더 항목의 인덱스 WHEN 라벨 생성 THEN 폴더 아이콘과 이름이 붙는다", func(t *testing.T) {
		if got := fileEntryLabel(1, entries); got != "📁 photos" {
			t.Errorf("got %q, 기대값 %q", got, "📁 photos")
		}
	})

	Scenario(t, "GIVEN 파일 항목의 인덱스 WHEN 라벨 생성 THEN 파일 아이콘과 이름이 붙는다", func(t *testing.T) {
		if got := fileEntryLabel(2, entries); got != "📄 readme.txt" {
			t.Errorf("got %q, 기대값 %q", got, "📄 readme.txt")
		}
	})
}

func TestDirExists(t *testing.T) {
	Scenario(t, "GIVEN 실제 존재하는 디렉터리 WHEN 확인 THEN true를 반환한다", func(t *testing.T) {
		if !dirExists(t.TempDir()) {
			t.Errorf("실제 존재하는 디렉터리인데 false가 나옴")
		}
	})

	// 부정 케이스: 존재하지 않는 경로.
	Scenario(t, "GIVEN 존재하지 않는 경로 WHEN 확인 THEN false를 반환한다 (부정 케이스)", func(t *testing.T) {
		if dirExists("/no/such/path/at/all/hopefully") {
			t.Errorf("존재하지 않는 경로인데 true가 나옴")
		}
	})
}

func TestDirsOnly(t *testing.T) {
	Scenario(t, "GIVEN 폴더와 파일이 섞인 목록 WHEN 폴더만 걸러냄 THEN 폴더만 남는다", func(t *testing.T) {
		entries := []fileEntry{
			{name: "photos", isDir: true},
			{name: "readme.txt", isDir: false},
			{name: "docs", isDir: true},
		}
		got := dirsOnly(entries)
		if len(got) != 2 {
			t.Fatalf("길이 = %d, 기대값 2 (%+v)", len(got), got)
		}
		for _, e := range got {
			if !e.isDir {
				t.Errorf("파일이 섞여 들어옴: %+v", e)
			}
		}
	})

	// 부정 케이스: 폴더가 하나도 없는 디렉터리.
	Scenario(t, "GIVEN 폴더가 하나도 없음 WHEN 폴더만 걸러냄 THEN 빈 목록을 반환한다 (부정 케이스)", func(t *testing.T) {
		entries := []fileEntry{{name: "a.txt", isDir: false}}
		if got := dirsOnly(entries); len(got) != 0 {
			t.Errorf("빈 목록을 기대했는데 %+v", got)
		}
	})
}
