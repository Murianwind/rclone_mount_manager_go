# RcloneManager

Windows용 rclone 마운트 관리 트레이 앱. [Python/Tkinter 버전](https://github.com/Murianwind/rclone_mount_manager)을 Go + Fyne으로 다시 만든 프로젝트입니다.

[![CI](https://github.com/Murianwind/rclone_mount_manager_go/actions/workflows/ci.yml/badge.svg)](https://github.com/Murianwind/rclone_mount_manager_go/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/Murianwind/rclone_mount_manager_go/graph/badge.svg)](https://codecov.io/gh/Murianwind/rclone_mount_manager_go)
[![Go Version](https://img.shields.io/badge/Go-1.22-00ADD8?logo=go&logoColor=white)](go.mod)
[![Latest Release](https://img.shields.io/github/v/release/Murianwind/rclone_mount_manager_go?include_prereleases)](https://github.com/Murianwind/rclone_mount_manager_go/releases)

## 주요 기능

- rclone 리모트를 드라이브 문자로 마운트/언마운트 (그레이스풀 종료 → WinFsp 마운트포인트가 깨끗하게 해제됨)
- rclone.conf에서 원본 리모트 가져오기 → 바로 마운트로 등록
- 마운트/원본 목록 순서 변경, 시작 시 자동 마운트, 시작프로그램 등록
- 네트워크 단절 시 마운트된 드라이브 자동 해제, 재연결 시 자동 마운트 항목 재마운트
- 마운트 실패 시 rclone 오류 메시지를 그대로 보여주는 상세 오류창
- 앱 자체 자동 업데이트, rclone.exe 자체 설치/업데이트 (둘 다 GitHub Release 기반)
- 트레이 아이콘에서 마운트 목록 확인 및 개별 마운트/해제
- 앱 종료 시 활성 마운트를 모두 안전하게 해제한 뒤 종료 (드라이브가 고아 프로세스로 남지 않음)

## 요구 사항

- Windows 10/11
- [rclone](https://rclone.org/) 실행 파일 (앱에서 자체적으로 다운로드/업데이트 가능)
- [WinFsp](https://winfsp.dev/) — rclone 마운트에 필요

## 다운로드

[Releases](https://github.com/Murianwind/rclone_mount_manager_go/releases)에서 최신 `RcloneManager.zip`을 받아 압축을 풀고 `RcloneManager.exe`를 실행하세요.

## 개발

```
go build -ldflags -H=windowsgui -o RcloneManager.exe ./cmd/gui
```

`-H=windowsgui`를 빼면 콘솔 창이 함께 뜹니다(디버깅 시에는 유용할 수 있습니다).

### 테스트

```
go test ./internal/... -v -cover   # 엔진 로직 (커버리지 대상)
go test ./cmd/gui/... -v           # GUI의 순수 로직 헬퍼
```

모든 테스트는 `GIVEN/WHEN/THEN` 시나리오 + 키워드 주도 방식으로 작성돼 있어, `-v` 출력 자체가 동작 명세 역할을 합니다.

## 프로젝트 구조

```
internal/engine/   순수 로직 (설정 파일 I/O, 마운트 커맨드 조립, 버전 비교/파싱,
                    rclone.conf 파싱, 다운로드/업데이트, Windows 시작프로그램 등록)
                    — OS/GUI 의존성이 없어 어떤 플랫폼에서도 빌드·테스트 가능

cmd/gui/           Fyne 기반 Windows 데스크톱 UI. 관심사별로 파일이 나뉘어 있습니다:
  main.go            진입점
  model.go           앱 상태, 공용 헬퍼
  rows.go            원본/마운트 통합 테이블 행 모델
  layout.go          창 레이아웃
  table.go           마운트 목록 표
  dialogs.go         추가/편집/삭제/오류 다이얼로그
  confimport.go      rclone.conf 가져오기
  reorder.go         목록 순서 변경
  mount.go           마운트/언마운트 프로세스 관리, 자동 마운트, 네트워크 모니터
  update.go          앱 자체 업데이트
  rcloneupdate.go    rclone.exe 자체 업데이트
  tray.go            트레이 아이콘/메뉴
```

## 라이선스

라이선스 파일이 아직 없습니다.
