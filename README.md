# RcloneManager

Windows용 rclone 마운트 관리 트레이 앱입니다. rclone 리모트를 드라이브 문자(예: `Z:`)로 마운트/해제하고, 시스템 트레이에서 편하게 관리할 수 있게 해줍니다.

[![CI](https://github.com/Murianwind/rclone_mount_manager_go/actions/workflows/ci.yml/badge.svg)](https://github.com/Murianwind/rclone_mount_manager_go/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/Murianwind/rclone_mount_manager_go/graph/badge.svg)](https://codecov.io/gh/Murianwind/rclone_mount_manager_go)
[![Latest Release](https://img.shields.io/github/v/release/Murianwind/rclone_mount_manager_go?include_prereleases)](https://github.com/Murianwind/rclone_mount_manager_go/releases)

## 이런 분께 필요합니다

- 구글 드라이브, 원드라이브 같은 클라우드 저장소를 로컬 드라이브처럼 쓰고 싶은 분
- rclone은 알지만 매번 명령어를 치는 게 번거로운 분
- PC를 켤 때마다 자동으로 클라우드 드라이브가 마운트돼 있었으면 하는 분

## 다운로드 및 설치

1. [Releases](https://github.com/Murianwind/rclone_mount_manager_go/releases) 페이지에서 최신 `RcloneManager.zip`을 받습니다.
2. 원하는 폴더에 압축을 풀고 `RcloneManager.exe`를 실행합니다.
3. [WinFsp](https://winfsp.dev/)가 설치돼 있지 않다면 먼저 설치해 주세요 — rclone이 클라우드 저장소를 실제 드라이브처럼 보이게 하는 데 필요한 드라이버입니다.
4. rclone 실행 파일이 없다면, 프로그램 상단의 rclone 버전 표시를 클릭하면 자동으로 다운로드/설치할 수 있습니다.

## 사용법

**처음 설정할 때**
1. 이미 rclone을 쓰고 계셨다면(`rclone.conf`가 있다면), **conf 가져오기** 버튼으로 기존 리모트 목록을 한 번에 불러올 수 있습니다.
2. 불러온 리모트 옆의 **가져오기** 버튼을 누르면 마운트 설정 추가 화면이 뜹니다 — 드라이브 문자, 캐시 옵션 등을 정하고 저장하세요.
3. 리모트가 없다면 **추가** 버튼으로 직접 새 마운트를 등록할 수 있습니다.

**평소 사용**
- 목록에서 **마운트**/**해제** 버튼으로 개별 드라이브를 켜고 끕니다.
- **자동** 체크박스를 켠 마운트는 프로그램 시작 시 자동으로 연결됩니다.
- 창을 닫아도 프로그램은 종료되지 않고 **트레이(작업 표시줄 오른쪽 아이콘)**로 들어갑니다. 트레이 아이콘을 좌클릭하면 창이 다시 열리고, 우클릭하면 마운트 목록을 바로 켜고 끌 수 있는 메뉴가 뜹니다.
- 완전히 종료하려면 트레이 메뉴의 **종료**를 누르세요 — 이때 켜져 있던 마운트가 전부 안전하게 해제된 뒤 종료됩니다.

**옵션 (창 상단 체크박스)**
- **시작 시 자동 실행**: Windows 시작 시 프로그램도 함께 실행
- **시작 시 자동 마운트**: 프로그램이 켜지면 "자동" 체크된 드라이브를 바로 마운트
- **시작 시 트레이로 최소화**: 실행하자마자 창을 띄우지 않고 바로 트레이로

## 자동 업데이트

- 프로그램 상단의 버전 배지, rclone 버전 표시를 각각 클릭하면 새 버전이 있는지 바로 확인합니다.
- 새 버전이 있으면 알림이 뜨고, 확인을 누르면 자동으로 받아서 적용 후 재시작됩니다.
- 백그라운드에서도 주기적으로 조용히 확인하며, 새 버전을 찾으면 (트레이에 있어도) 창을 띄워 알려줍니다.

## 문제가 생겼다면

- 프로그램 폴더의 `RcloneManager.log` 파일에 마운트/업데이트 관련 기록이 남습니다. 문제가 재현되면 이 파일을 확인해 주세요.
- 마운트가 실패하면 rclone이 출력한 오류 메시지를 그대로 보여주는 창이 뜹니다.
- 버그 제보나 문의는 창 상단의 **!** 버튼(또는 [Issues](https://github.com/Murianwind/rclone_mount_manager_go/issues/new)) 페이지를 이용해 주세요.

## 요구 사항

- Windows 10/11
- [WinFsp](https://winfsp.dev/)
- [rclone](https://rclone.org/) (프로그램에서 자체적으로 받을 수 있음)

---

## 개발자용 정보

이 프로젝트는 [Python/Tkinter 버전](https://github.com/Murianwind/rclone_mount_manager)을 Go + Fyne으로 다시 만든 것입니다.

### 빌드

```
go build -ldflags -H=windowsgui -o RcloneManager.exe ./cmd/gui
```

`-H=windowsgui`를 빼면 콘솔 창이 함께 뜹니다(디버깅 시 유용할 수 있습니다).

### 테스트

```
go test ./internal/... -v -cover   # 엔진 로직 (커버리지 대상)
go test ./cmd/gui/... -v           # GUI의 순수 로직 헬퍼
```

모든 테스트는 `GIVEN/WHEN/THEN` 시나리오 + 키워드 주도 방식으로 작성돼 있어, `-v` 출력 자체가 동작 명세 역할을 합니다.

### 프로젝트 구조

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
