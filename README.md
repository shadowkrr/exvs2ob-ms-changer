# ms-changer: GUI & CLI Edition

A utility for modifying the in-game Mobile Suit of a Windows-based arcade client via memory editing.

This project includes both:
- A **GUI** (built with Fyne) for easy selection
- A **CLI** (invoked by GUI) that performs memory write operations

---

## 📁 Files

| File                     | Description                                  |
|--------------------------|----------------------------------------------|
| `units.csv`              | CSV list of units (`id,name,unitID`)         |
| `ms-changer.go`          | Standalone CLI for writing memory            |
| `ms-changer-gui.go`      | Fyne-based GUI frontend                      |
| `ms-changer-gui-cli.go`  | CLI backend invoked by GUI                   |
| `README.md`              | This documentation                           |

---

## ⚙️ Build Instructions

### 🔲 GUI (no console window)

```bash
go build -ldflags="-H windowsgui" -o ms-changer-gui.exe ms-changer-gui.go
```

### 💻 CLI

```bash
go build -o ms-changer.exe ms-changer.go
go build -o ms-changer-gui-cli.exe ms-changer-gui-cli.go
```

> 🔔 Make sure all `.exe` and `.csv` files are in the same directory.

---

## 📄 CSV Format

Example `units.csv`:

```csv
id,name,unitID
1,Gundam,1001001
2,Char's Gelgoog,1002001
3,Z Gundam,2001001
```

- `id`: Internal ID for menu order
- `name`: Displayed in GUI
- `unitID`: Value written into memory

---

## 🕹️ How It Works

1. GUI waits for the target game process (`vsac27_Release_CLIENT.exe`)
2. User selects a Mobile Suit via the GUI (radio buttons)
3. GUI executes `ms-changer-gui-cli.exe` with the selected `unitID`
4. CLI locates the process and writes to memory

---

## ⚠️ Important Notes

- ✅ Run as **Administrator**
- 🕒 GUI continuously checks for the game process until it's found
- 🛠️ For **educational and personal use only**

---
