# PAL Kit 환경 관리 가이드

> 다중 PC 환경에서 동일한 프로젝트를 관리하는 방법

---

## 개요

PAL Kit의 환경 관리 기능을 통해:

- **다중 PC 지원**: 집, 회사 등 다른 PC에서 동일한 프로젝트 작업
- **경로 유연성**: 프로젝트/Docs가 다른 위치에 있어도 동일한 ID로 관리
- **자동 설정 생성**: 환경 전환 시 `.claude/settings.local.json` 자동 업데이트
- **MCP 서버 연동**: Docs vault 경로를 MCP 서버에 자동 반영

---

## 핵심 개념

### 환경 (Environment)

PC마다 하나의 환경을 등록합니다. hostname으로 자동 식별됩니다.

```
~/.pal/environments.yaml

environments:
  home-mac:
    hostname: MacBook-Pro.local
    paths:
      workspace: ~/playground
      claude_data: ~/.claude
    projects:
      pal-kit:
        path: ~/playground/CodeSpace/pal-kit
        docs: mcp-docs
    docs:
      mcp-docs:
        path: ~/playground/mcp-docs

  work-pc:
    hostname: WORK-PC
    paths:
      workspace: /Users/john/dev
    projects:
      pal-kit:
        path: /Users/john/dev/pal-kit
        docs: work-vault
    docs:
      work-vault:
        path: /Users/john/Documents/vault
```

### 프로젝트 매핑

동일한 프로젝트 ID가 환경마다 다른 경로를 가질 수 있습니다.

```
프로젝트 ID: pal-kit
  - home-mac: ~/playground/CodeSpace/pal-kit
  - work-pc:  /Users/john/dev/pal-kit
```

### Docs Vault

Obsidian vault 등 문서 저장소를 환경별로 관리합니다.

```
Docs ID: mcp-docs
  - home-mac: ~/playground/mcp-docs
  - work-pc:  /Users/john/Documents/vault
```

---

## 초기 설정

### 1. 환경 등록

```bash
# 현재 PC를 환경으로 등록 (hostname 자동 사용)
pal env setup

# 또는 이름 지정
pal env setup home-mac
```

### 2. Docs Vault 추가

```bash
# Docs vault 등록
pal env add-docs mcp-docs --path ~/playground/mcp-docs

# 확인
pal env show
```

### 3. 프로젝트 추가

```bash
# 프로젝트 등록 (Docs 연결 포함)
pal env add-project pal-kit --path ~/playground/CodeSpace/pal-kit --docs mcp-docs

# 또는 나중에 연결
pal env add-project pal-kit --path ~/playground/CodeSpace/pal-kit
pal env link pal-kit mcp-docs
```

### 4. 설정 적용

```bash
# 모든 프로젝트에 settings.local.json 생성
pal env apply

# 특정 프로젝트만
pal env apply pal-kit
```

---

## 일상 사용

### 새 PC에서 시작

```bash
# 1. PAL Kit 설치
go install github.com/n0roo/pal-kit/cmd/pal@latest

# 2. 환경 등록
pal env setup work-pc

# 3. Docs vault 추가 (이 PC의 경로)
pal env add-docs mcp-docs --path /Users/john/Documents/vault

# 4. 프로젝트 추가 (이 PC의 경로)
pal env add-project pal-kit --path /Users/john/dev/pal-kit --docs mcp-docs

# 5. 설정 적용
pal env apply
```

### 환경 전환 (자동)

```bash
# 현재 PC에 맞는 환경 자동 감지 및 전환
pal env detect --switch
```

### 환경 확인

```bash
# 모든 환경 목록
pal env list

# 현재 환경 상세
pal env show

# 특정 환경 상세
pal env show work-pc
```

---

## 생성되는 파일

### settings.local.json

`pal env apply` 실행 시 각 프로젝트의 `.claude/settings.local.json`이 생성됩니다:

```json
{
  "mcpServers": {
    "pa-retriever": {
      "command": "npx",
      "args": ["-y", "@anthropics/mcp-pa-retriever", "/Users/n0roo/playground/mcp-docs"]
    }
  },
  "additionalDirectories": [
    "/Users/n0roo/playground/mcp-docs"
  ]
}
```

이 파일은 `.gitignore`에 추가하여 각 PC마다 별도로 관리합니다.

### environments.yaml

`~/.pal/environments.yaml`에 모든 환경 정보가 저장됩니다.

---

## Claude와의 대화 예시

### 프로젝트 시작 시

```
User: pal-kit 프로젝트 작업할게

Claude: 현재 환경을 확인합니다.
[pal env current 실행]

현재 환경: home-mac
프로젝트 경로: ~/playground/CodeSpace/pal-kit
연결된 Docs: mcp-docs (~/playground/mcp-docs)

작업을 시작합니다. 무엇을 도와드릴까요?
```

### 환경 전환 필요 시

```
User: 회사 PC에서 작업 이어서 하려고 해

Claude: 환경을 전환합니다.
[pal env detect --switch 실행]

환경 전환 완료: work-pc
settings.local.json 업데이트:
  ✓ /Users/john/dev/pal-kit/.claude/settings.local.json

이제 work-pc 환경에서 작업할 준비가 되었습니다.
```

### 새 프로젝트 추가 시

```
User: 새 프로젝트 k-esg 추가해줘. 경로는 ~/projects/k-esg이고 mcp-docs랑 연결해줘

Claude: 프로젝트를 추가합니다.
[pal env add-project k-esg --path ~/projects/k-esg --docs mcp-docs 실행]

프로젝트 추가 완료: k-esg → ~/projects/k-esg
연결된 Docs: mcp-docs

[pal env apply k-esg 실행]

생성 완료: ~/projects/k-esg/.claude/settings.local.json
```

---

## 명령어 요약

| 명령어 | 설명 |
|--------|------|
| `pal env setup [name]` | 현재 PC를 환경으로 등록 |
| `pal env list` | 등록된 환경 목록 |
| `pal env current` | 현재 환경 표시 |
| `pal env show [name]` | 환경 상세 정보 |
| `pal env switch <name>` | 환경 전환 + 설정 자동 적용 |
| `pal env detect [--switch]` | 환경 자동 감지 |
| `pal env add-docs <id> --path <path>` | Docs vault 추가 |
| `pal env remove-docs <id>` | Docs vault 제거 |
| `pal env add-project <id> --path <path> [--docs <docs-id>]` | 프로젝트 추가 |
| `pal env remove-project <id>` | 프로젝트 제거 |
| `pal env link <project-id> <docs-id>` | 프로젝트-Docs 연결 |
| `pal env apply [project-id]` | settings.local.json 생성 |
| `pal env delete <name>` | 환경 삭제 |

---

## 주의사항

1. **settings.local.json은 gitignore에 추가**
   ```gitignore
   .claude/settings.local.json
   ```

2. **환경 전환 시 자동으로 settings 재생성**
   - `pal env switch` 실행 시 모든 프로젝트의 settings.local.json이 재생성됩니다

3. **Docs vault는 환경별로 등록**
   - 같은 Docs ID라도 환경마다 다른 경로를 가질 수 있습니다
   - 동기화는 Git 등 별도 도구로 관리하세요

4. **현재 환경 삭제 불가**
   - 다른 환경으로 먼저 전환 후 삭제하세요

---

## 문제 해결

### 환경 감지 실패

```bash
# hostname 확인
hostname

# 등록된 환경의 hostname과 비교
pal env list
pal env show <name>

# 수동 전환
pal env switch <name>
```

### settings.local.json 재생성

```bash
# 강제 재생성
pal env apply
```

### 프로젝트 경로 변경

```bash
# 기존 제거 후 재등록
pal env remove-project pal-kit
pal env add-project pal-kit --path /new/path --docs mcp-docs
pal env apply pal-kit
```
