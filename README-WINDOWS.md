# 🚀 SuperPlane - Ghid Rapid pentru Windows

## Pași pentru a porni aplicația

### 1. Pornește Docker Desktop
- Caută "Docker Desktop" în meniul Start Windows
- Dă click pentru a porni aplicația
- **Așteaptă** până când icon-ul din system tray devine verde (Docker este gata)

### 2. Pornește SuperPlane
Dublu-click pe:
```
start-superplane.bat
```

Scriptul va:
- ✅ Verifica dacă Docker rulează
- ✅ Porni toate containerele necesare
- ✅ Aștepta ca aplicația să fie gata
- ✅ Deschide automat browserul la http://localhost:8000

### 3. Accesează aplicația
Browser-ul se va deschide automat sau navighează manual la:
```
http://localhost:8000
```

## 📝 Scripturi disponibile

| Script | Descriere |
|--------|-----------|
| `start-superplane.bat` | Pornește SuperPlane |
| `stop-superplane.bat` | Oprește SuperPlane |
| `logs-superplane.bat` | Vezi log-urile în timp real |
| `test-listpipelines.bat` | Rulează testele pentru List Pipelines |

## 🧪 Testarea componentei List Pipelines

După ce aplicația pornește, urmează acești pași:

### Pas 1: Configurează integrarea Semaphore
1. Click pe **"Integrations"** în meniul din stânga
2. Click pe **"+ Add Integration"**
3. Selectează **"Semaphore"**
4. Completează:
   - **Organization URL**: `https://your-org.semaphoreci.com`
   - **API Token**: token-ul tău Semaphore
5. Click **"Save"**

### Pas 2: Creează un workflow
1. Click pe **"Canvases"** în meniul din stânga
2. Click pe **"+ New Canvas"**
3. Dă un nume workflow-ului (ex: "Test List Pipelines")

### Pas 3: Adaugă componenta List Pipelines
1. Click pe butonul **"+"** pentru a adăuga o componentă
2. Caută **"List Pipelines"** în secțiunea **Semaphore**
3. Configurează:
   - **Project**: Selectează un project Semaphore din dropdown
   - **Branch Name** (opțional): ex: `main`, `develop`
   - **Limit** (opțional): ex: `10` (default: 30, max: 100)
   - Alte filtre opționale după necesitate

### Pas 4: Rulează și verifică rezultatele
1. Click pe **"Run"** pentru a executa workflow-ul
2. Vezi rezultatele în panoul de execuție
3. Ar trebui să vezi lista de pipeline-uri returnată

### Rezultat așteptat:
```json
[
  {
    "ppl_id": "pipeline-id",
    "wf_id": "workflow-id", 
    "name": "Pipeline Name",
    "state": "done",
    "result": "passed",
    "created_at": "2024-01-15T10:30:00Z",
    "done_at": "2024-01-15T10:45:00Z",
    "branch_name": "main",
    "yml_file_path": ".semaphore/semaphore.yml"
  }
]
```

## 🔧 Comenzi Docker directe (dacă vrei să le folosești manual)

### Pornește aplicația:
```cmd
docker compose -f docker-compose.dev.yml up -d
```

### Vezi status-ul containerelor:
```cmd
docker compose -f docker-compose.dev.yml ps
```

### Vezi log-urile:
```cmd
docker compose -f docker-compose.dev.yml logs -f
```

### Oprește aplicația:
```cmd
docker compose -f docker-compose.dev.yml down
```

### Rebuild containers (după modificări de cod):
```cmd
docker compose -f docker-compose.dev.yml up -d --build
```

## ⚠️ Troubleshooting

### Docker nu pornește
- Verifică dacă ai WSL 2 instalat (necesar pentru Docker pe Windows)
- Restart la Windows
- Reinstalează Docker Desktop

### Portul 8000 este ocupat
```cmd
netstat -ano | findstr :8000
taskkill /PID <PID> /F
```

### Containerele nu pornesc
```cmd
docker compose -f docker-compose.dev.yml down -v
docker compose -f docker-compose.dev.yml up -d --build
```

### Vrei să vezi ce se întâmplă în container
```cmd
docker compose -f docker-compose.dev.yml exec app /bin/bash
```

## 📚 Resurse

- **Documentație**: https://docs.superplane.com
- **Discord**: https://discord.gg/KC78eCNsnw
- **GitHub**: https://github.com/superplanehq/superplane

## ✅ Componenta List Pipelines - Detalii

### Filtre disponibile:
- **Project** (required): Semaphore project ID sau name
- **Branch Name**: Filtrează după branch (ex: `main`, `develop`)
- **YML File Path**: Filtrează după fișier pipeline (ex: `.semaphore/semaphore.yml`)
- **Created After**: Pipeline-uri create după această dată
- **Created Before**: Pipeline-uri create înainte de această dată
- **Done After**: Pipeline-uri terminate după această dată
- **Done Before**: Pipeline-uri terminate înainte de această dată
- **Limit**: Număr maxim de pipeline-uri (default: 30, max: 100)

### Use Cases:
✅ Dashboard cu status-ul recent al pipeline-urilor  
✅ Găsirea celui mai recent pipeline pentru un branch  
✅ Iterare prin pipeline-uri pentru acțiuni automate  
✅ Raportare asupra pipeline-urilor eșuate  

---

**Data implementării:** 5 Februarie 2026  
**Status:** ✅ Complet și gata de testare  
