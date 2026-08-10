# rclone-manager-go — Phase 1 (engine only)

Go 포팅 1단계: GUI/OS 의존성 없는 순수 로직만 옮긴 상태입니다.
`internal/engine` 패키지에 Python `rclone_manager.py`의 다음 함수들을
동일한 동작으로 이식했습니다.

| Python                          | Go                              |
|----------------------------------|----------------------------------|
| `normalize_flags`                | `engine.NormalizeFlags`          |
| `_get_volname`                   | `engine.GetVolname`              |
| `build_cmd`                      | `engine.BuildCmd`                |
| `_ver_tuple`                     | `engine.VerTuple` / `CompareVersions` |
| `load_config` / `save_config`    | `engine.Store.Load` / `.Save`    |
| `is_internet_available`          | `engine.IsInternetAvailable`     |

기존 Python 테스트(`test_scenario_02~07d`, `60~68`, `71~75`)에 대응하는
Go 테스트를 작성했고, 로컬에서 `go test ./... -cover` 결과 **24개 전부
통과, 커버리지 90.7%**, `GOOS=windows go build`도 정상 확인했습니다.

## 아직 안 옮긴 것 (Phase 2/3)

- **GUI**: Tkinter 대체할 Go 프레임워크 선택 필요 (Fyne / Walk(win32 네이티브) / 트레이만 Go + 별도 웹 설정 화면 등). 이건 완전히 다른 논의라 의도적으로 미룸.
- **rclone 다운로드/버전 체크** (`download_rclone`, GitHub Releases 조회): 엔진에 포함 가능하지만 실제 네트워크 I/O라 우선순위 낮춤.
- **앱 자체 업데이터** (`download_app_release`)
- **Windows 시작프로그램 등록** (`is_startup_enabled` / `set_startup`, `winreg` 기반): Go에서는 `golang.org/x/sys/windows/registry` 사용 가능, Windows 빌드 태그 필요.
- **트레이 메뉴 / 네트워크 단절 자동 언마운트 모니터**
- **로그 rotate** (`write_log`): 로직 자체는 간단해서 원하면 바로 추가 가능.

## 실행

```
go vet ./...
go test ./... -v -cover
GOOS=windows GOARCH=amd64 go build ./...
```
