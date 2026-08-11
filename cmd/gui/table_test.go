package main

import "testing"

func TestDisplayDrive(t *testing.T) {
	Scenario(t, "GIVEN 드라이브 문자가 비어있음 WHEN 표시 텍스트 결정 THEN '(자동)'으로 보여준다", func(t *testing.T) {
		if got := displayDrive(""); got != "(자동)" {
			t.Errorf("displayDrive(\"\") = %q, 기대값 %q", got, "(자동)")
		}
	})
	Scenario(t, "GIVEN 드라이브 문자가 공백뿐임 WHEN 표시 텍스트 결정 THEN '(자동)'으로 보여준다", func(t *testing.T) {
		if got := displayDrive("  "); got != "(자동)" {
			t.Errorf("displayDrive(\"  \") = %q, 기대값 %q", got, "(자동)")
		}
	})
	Scenario(t, "GIVEN 드라이브 문자가 지정돼 있음 WHEN 표시 텍스트 결정 THEN 그 값을 그대로 보여준다", func(t *testing.T) {
		if got := displayDrive("E:"); got != "E:" {
			t.Errorf("displayDrive(\"E:\") = %q, 기대값 %q", got, "E:")
		}
	})
}

func TestStatusLabel(t *testing.T) {
	Scenario(t, "GIVEN 마운트가 실행 중임 WHEN 상태 라벨 결정 THEN '연결됨'을 보여준다", func(t *testing.T) {
		if got := statusLabel(true); got != "연결됨" {
			t.Errorf("statusLabel(true) = %q, 기대값 %q", got, "연결됨")
		}
	})
	Scenario(t, "GIVEN 마운트가 실행 중이 아님 WHEN 상태 라벨 결정 THEN '해제됨'을 보여준다", func(t *testing.T) {
		if got := statusLabel(false); got != "해제됨" {
			t.Errorf("statusLabel(false) = %q, 기대값 %q", got, "해제됨")
		}
	})
}

func TestToggleLabel(t *testing.T) {
	Scenario(t, "GIVEN 마운트가 실행 중임 WHEN 토글 버튼 라벨 결정 THEN '해제'를 보여준다", func(t *testing.T) {
		if got := toggleLabel(true); got != "해제" {
			t.Errorf("toggleLabel(true) = %q, 기대값 %q", got, "해제")
		}
	})
	Scenario(t, "GIVEN 마운트가 실행 중이 아님 WHEN 토글 버튼 라벨 결정 THEN '마운트'를 보여준다", func(t *testing.T) {
		if got := toggleLabel(false); got != "마운트" {
			t.Errorf("toggleLabel(false) = %q, 기대값 %q", got, "마운트")
		}
	})
}

func TestToggleSelection(t *testing.T) {
	Scenario(t, "GIVEN 선택된 행이 없음(-1) WHEN 어떤 행을 클릭 THEN 그 행이 선택된다", func(t *testing.T) {
		if got := toggleSelection(-1, 2); got != 2 {
			t.Errorf("got %d, 기대값 2", got)
		}
	})

	Scenario(t, "GIVEN 이미 선택된 행을 WHEN 다시 클릭 THEN 선택이 해제된다(-1)", func(t *testing.T) {
		if got := toggleSelection(2, 2); got != -1 {
			t.Errorf("got %d, 기대값 -1", got)
		}
	})

	Scenario(t, "GIVEN 다른 행이 선택된 상태에서 WHEN 새 행을 클릭 THEN 선택이 그 새 행으로 옮겨간다", func(t *testing.T) {
		if got := toggleSelection(1, 3); got != 3 {
			t.Errorf("got %d, 기대값 3", got)
		}
	})
}
