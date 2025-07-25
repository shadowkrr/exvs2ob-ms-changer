# ms-changer: GUI & CLI Edition

A utility for modifying the in-game Mobile Suit of a Windows-based arcade client via memory editing.

This project includes both:
- A **GUI** (built with Fyne) for easy selection
- A **CLI** (invoked by GUI) that performs memory write operations

---

## 📁 Files

| File                     | Description                                  |
|--------------------------|----------------------------------------------|
| `units.csv`              | CSV list of units (`id,title,name,value`)    |
| `ms-changer.go`          | CLI tool for direct memory manipulation      |
| `ms-changer-gui.go`      | GUI frontend written in Fyne                 |
| `ms-changer-gui-cli.go`  | CLI called by the GUI for memory writing     |
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
id,title,name,value
1,Mobile Suit Gundam,Gundam,1001001
2,Mobile Suit Gundam,Char's Gelgoog,1002001
3,Mobile Suit Zeta Gundam,Z Gundam,2001001
```

- `id`: Internal identifier for sorting
- `title`: Series title of the Mobile Suit
- `name`: Mobile Suit NameName of the Mobile Suit
- `value`: Memory value written to the process

---

## 🕹️ How It Works

1. GUI waits for the target game process (`vsac27_Release_CLIENT.exe`)
2. User selects a Mobile Suit via the GUI (radio buttons)
3. GUI executes `ms-changer-gui-cli.exe` with the selected `value`
4. CLI locates the process and writes to memory

---

## ⚠️ Important Notes

- ✅ Run as **Administrator**
- 🕒 GUI continuously checks for the game process until it's found
- 🛠️ For **educational and personal use only**

---
