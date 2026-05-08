from __future__ import annotations

import csv
import gc
import json
import os
import re
import subprocess
import sys
import tempfile
import time
import unicodedata
from dataclasses import dataclass
from datetime import datetime
from pathlib import Path
from tkinter import Tk, filedialog, messagebox, simpledialog
import tkinter as tk

from openpyxl import Workbook
from openpyxl.styles import Alignment, Border, Font, PatternFill, Side
from openpyxl.worksheet.table import Table
from openpyxl.utils import column_index_from_string, get_column_letter

try:
    from tkinterdnd2 import DND_FILES, TkinterDnD
except Exception:
    DND_FILES = None
    TkinterDnD = None


EXCLUDED_JOURNALS = {"AT", "AC", "OD", "CA", "AN", "RV"}
MOVEMENT_CHARS_PER_LINE = 64
BASE_ROW_HEIGHT = 22
WRAPPED_LINE_HEIGHT = 18
ROW_HEIGHT_PADDING = 4
MAX_ROW_HEIGHT = 84
REQUIRED_COLUMNS = {
    "libelle_du_compte": "Libellé du compte",
    "date": "Date",
    "journal": "Journal",
    "n_de_piece": "N° de pièce",
    "libelle_mouvement": "Libellé mouvement",
    "montant_debit": "Montant Débit",
    "montant_credit": "Montant Crédit",
}


@dataclass
class OutputInfo:
    source: Path
    excel_output: Path | None
    pdf_output: Path | None
    row_count: int
    amount_label: str
    total_amount: float
    treatment_label: str


@dataclass(frozen=True)
class OutputFormats:
    excel: bool
    pdf: bool

    @property
    def has_any(self) -> bool:
        return self.excel or self.pdf


@dataclass(frozen=True)
class Treatment:
    key: str
    label: str
    output_label: str
    sheet_title: str
    table_name: str
    amount_label: str


PAYMENTS_WITHOUT_INVOICE = Treatment(
    key="payments",
    label="Paiements sans facture",
    output_label="PAIEMENTS SANS FACTURE",
    sheet_title="Paiements sans facture",
    table_name="PaiementsSansFacture",
    amount_label="total paiements",
)
INVOICES_WITHOUT_PAYMENT = Treatment(
    key="invoices",
    label="Factures sans paiements",
    output_label="FACTURES SANS PAIEMENTS",
    sheet_title="Factures sans paiements",
    table_name="FacturesSansPaiements",
    amount_label="total factures",
)
ALL_TREATMENTS = (PAYMENTS_WITHOUT_INVOICE, INVOICES_WITHOUT_PAYMENT)
EXCEL_ONLY = OutputFormats(excel=True, pdf=False)
EXCEL_AND_PDF = OutputFormats(excel=True, pdf=True)


def settings_path() -> Path:
    base = Path(os.environ.get("APPDATA") or Path.home() / "AppData" / "Roaming")
    return base / "RelanceTiime" / "settings.json"


def load_saved_formats(default_formats: OutputFormats, invalid_fallback: OutputFormats = EXCEL_ONLY) -> OutputFormats:
    path = settings_path()
    if not path.exists():
        return default_formats
    try:
        data = json.loads(path.read_text(encoding="utf-8-sig"))
        formats = OutputFormats(excel=bool(data.get("excel")), pdf=bool(data.get("pdf")))
    except Exception:
        return default_formats
    if not formats.has_any:
        return invalid_fallback
    return formats


def save_formats(formats: OutputFormats) -> None:
    path = settings_path()
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(
        json.dumps({"excel": formats.excel, "pdf": formats.pdf}, indent=2),
        encoding="utf-8",
    )


def resource_path(name: str) -> Path:
    base = Path(getattr(sys, "_MEIPASS", Path(__file__).resolve().parent))
    return base / name


def norm(value: str) -> str:
    text = "".join(
        ch for ch in unicodedata.normalize("NFKD", str(value or ""))
        if not unicodedata.combining(ch)
    )
    return re.sub(r"[^a-z0-9]+", "_", text.lower()).strip("_")


def parse_money(value: str | None) -> float:
    text = str(value or "").replace("\xa0", " ").strip()
    if not text:
        return 0.0
    text = text.replace(" ", "").replace(",", ".")
    return float(text)


def format_amount(value: float) -> str:
    return f"{value:,.2f}".replace(",", " ").replace(".", ",")


def write_error_log(message: str) -> None:
    try:
        log_path = Path(tempfile.gettempdir()) / "relance_tiime_error.log"
        log_path.write_text(message, encoding="utf-8")
    except Exception:
        pass


def parse_date(value: str | None) -> datetime | None:
    text = str(value or "").strip()
    if not text:
        return None
    for fmt in ("%d/%m/%Y", "%Y-%m-%d", "%d-%m-%Y"):
        try:
            return datetime.strptime(text, fmt)
        except ValueError:
            pass
    raise ValueError(f"Date invalide : {text}")


def derive_client_year(csv_path: Path, root: Tk | None = None) -> tuple[str, str]:
    match = re.match(
        r"grand_livre_(\d{4})-\d{2}-\d{2}_(\d{4})-\d{2}-\d{2}_(.+)\.csv$",
        csv_path.name,
        flags=re.IGNORECASE,
    )
    if match:
        start_year, end_year, client_slug = match.groups()
        year = start_year if start_year == end_year else f"{start_year}-{end_year}"
        client = client_slug.replace("_", " ").replace("-", " ").upper().strip()
        return client, year

    if root is None:
        raise ValueError(
            "Nom de fichier non standard. Format attendu : "
            "grand_livre_2025-01-01_2025-12-31_nom_client.csv"
        )

    client = simpledialog.askstring("Client", "Nom du client :", parent=root)
    year = simpledialog.askstring("Année", "Année ou période :", parent=root)
    if not client or not year:
        raise ValueError("Client et année sont obligatoires pour ce fichier.")
    return client.upper().strip(), year.strip()


def unique_path(path: Path) -> Path:
    if not path.exists():
        return path
    stem, suffix = path.stem, path.suffix
    index = 2
    while True:
        candidate = path.with_name(f"{stem} - {index}{suffix}")
        if not candidate.exists():
            return candidate
        index += 1


def unique_output_paths(directory: Path, stem: str, formats: OutputFormats) -> tuple[Path | None, Path | None]:
    index = 1
    while True:
        suffix = "" if index == 1 else f" - {index}"
        excel_path = directory / f"{stem}{suffix}.xlsx" if formats.excel else None
        pdf_path = directory / f"{stem}{suffix}.pdf" if formats.pdf else None
        candidates = [path for path in (excel_path, pdf_path) if path is not None]
        if all(not path.exists() for path in candidates):
            return excel_path, pdf_path
        index += 1


def get_header_map(reader: csv.DictReader) -> dict[str, str]:
    if not reader.fieldnames:
        raise ValueError("Le CSV ne contient pas d'en-têtes.")
    header_map = {norm(header): header for header in reader.fieldnames}
    missing = [label for key, label in REQUIRED_COLUMNS.items() if key not in header_map]
    if missing:
        raise ValueError("Colonnes manquantes : " + ", ".join(missing))
    return header_map


def estimate_wrapped_lines(value: object, chars_per_line: int = MOVEMENT_CHARS_PER_LINE) -> int:
    text = str(value or "").strip()
    if not text:
        return 1

    total_lines = 0
    for raw_segment in text.splitlines() or [""]:
        words = raw_segment.split()
        if not words:
            total_lines += 1
            continue

        line_length = 0
        segment_lines = 1
        for word in words:
            word_length = len(word)
            if line_length == 0:
                line_length = word_length
            elif line_length + 1 + word_length <= chars_per_line:
                line_length += 1 + word_length
            else:
                segment_lines += 1
                line_length = word_length

            if line_length > chars_per_line:
                extra_lines = (line_length - 1) // chars_per_line
                segment_lines += extra_lines
                line_length = line_length % chars_per_line or chars_per_line

        total_lines += segment_lines

    return max(1, total_lines)


def row_height_for_movement(value: object) -> int:
    line_count = estimate_wrapped_lines(value)
    if line_count <= 1:
        return BASE_ROW_HEIGHT
    return min(MAX_ROW_HEIGHT, ROW_HEIGHT_PADDING + WRAPPED_LINE_HEIGHT * line_count)


def read_rows(csv_path: Path, treatment: Treatment) -> list[dict[str, object]]:
    with csv_path.open("r", encoding="cp1252", newline="") as handle:
        reader = csv.DictReader(handle, delimiter=";")
        header_map = get_header_map(reader)

        rows: list[dict[str, object]] = []
        for raw in reader:
            piece = str(raw.get(header_map["n_de_piece"], "") or "").strip()
            journal = str(raw.get(header_map["journal"], "") or "").strip().upper()
            debit = parse_money(raw.get(header_map["montant_debit"]))
            credit = parse_money(raw.get(header_map["montant_credit"]))
            date_value = parse_date(raw.get(header_map["date"]))
            base = {
                "Libellé du compte": str(raw.get(header_map["libelle_du_compte"], "") or "").strip(),
                "Date": date_value,
                "Journal": journal,
                "Libellé mouvement": str(raw.get(header_map["libelle_mouvement"], "") or "").strip(),
            }

            if treatment.key == "payments":
                if piece or journal in EXCLUDED_JOURNALS:
                    continue
                rows.append(
                    {
                        **base,
                        "Paiements": debit if debit != 0 else None,
                        "Rmbrsmts": credit if credit != 0 else None,
                        "Remarque": "Manque facture" if debit > 0 else "Manque avoir",
                    }
                )
            else:
                if journal not in {"AC", "AT"}:
                    continue
                rows.append(
                    {
                        **base,
                        "Factures": credit if credit != 0 else None,
                        "Avoirs": debit if debit != 0 else None,
                        "Remarque": "Facture sans paiement" if credit > 0 else "Avoir sans rmbrsmt",
                    }
                )

    rows.sort(
        key=lambda row: (
            str(row["Libellé du compte"]).upper(),
            row["Date"] or datetime.max,
            str(row["Journal"]),
            float(row.get("Paiements") or row.get("Factures") or 0),
            float(row.get("Rmbrsmts") or row.get("Avoirs") or 0),
        )
    )
    return rows


def write_workbook(
    rows: list[dict[str, object]],
    output_path: Path,
    treatment: Treatment,
) -> None:
    if treatment.key == "payments":
        include_refunds = any(row["Rmbrsmts"] not in (None, 0) for row in rows)
        headers = ["Libellé du compte", "Date", "Journal", "Libellé mouvement", "Paiements"]
        if include_refunds:
            headers.append("Rmbrsmts")
        headers.append("Remarque")
        amount_columns = [5] + ([6] if include_refunds else [])
    else:
        include_refunds = True
        headers = ["Libellé du compte", "Date", "Journal", "Libellé mouvement", "Factures", "Avoirs", "Remarque"]
        amount_columns = [5, 6]

    wb = Workbook()
    ws = wb.active
    ws.title = treatment.sheet_title
    last_col = len(headers)
    last_col_letter = get_column_letter(last_col)

    header_row = 1
    for column_index, header in enumerate(headers, start=1):
        ws.cell(header_row, column_index, header)

    for row_index, row in enumerate(rows, start=header_row + 1):
        if treatment.key == "payments":
            values = [
                row["Libellé du compte"],
                row["Date"],
                row["Journal"],
                row["Libellé mouvement"],
                row["Paiements"],
            ]
            if include_refunds:
                values.append(row["Rmbrsmts"])
            values.append(row["Remarque"])
        else:
            values = [
                row["Libellé du compte"],
                row["Date"],
                row["Journal"],
                row["Libellé mouvement"],
                row["Factures"],
                row["Avoirs"],
                row["Remarque"],
            ]
        for column_index, value in enumerate(values, start=1):
            ws.cell(row_index, column_index, value)

    last_row = max(header_row, header_row + len(rows))
    table_end_row = max(last_row, header_row + 1)
    table_ref = f"A{header_row}:{last_col_letter}{table_end_row}"
    table = Table(displayName=treatment.table_name, ref=table_ref)
    ws.add_table(table)

    header_fill = PatternFill("solid", fgColor="1F4E78")
    header_font = Font(name="Segoe UI", size=11, color="FFFFFF", bold=True)
    thin = Side(style="thin", color="808080")
    full_border = Border(left=thin, right=thin, top=thin, bottom=thin)
    for cell in ws[header_row]:
        cell.fill = header_fill
        cell.font = header_font
        cell.alignment = Alignment(horizontal="center", vertical="center")
        cell.border = full_border

    for row in ws.iter_rows(min_row=header_row + 1, max_row=last_row):
        for cell in row:
            horizontal = "left"
            if cell.column in (2, 3):
                horizontal = "center"
            elif cell.column in amount_columns:
                horizontal = "center"
            elif cell.column == last_col:
                horizontal = "center"
            cell.alignment = Alignment(horizontal=horizontal, vertical="center", wrap_text=cell.column == 4)
            cell.font = Font(name="Segoe UI", size=11)
            if cell.value not in (None, ""):
                cell.border = full_border

    widths = {
        "A": 24,
        "B": 12,
        "C": 9,
        "D": 76,
        "E": 16,
        "F": 16 if include_refunds else 18,
        "G": 18,
    }
    for col, width in widths.items():
        if last_col >= column_index_from_string(col):
            ws.column_dimensions[col].width = width

    for row_index in range(header_row + 1, last_row + 1):
        ws.cell(row_index, 2).number_format = "dd/mm/yyyy"
        for column_index in amount_columns:
            ws.cell(row_index, column_index).number_format = '#,##0.00 "€"'
        movement = str(ws.cell(row_index, 4).value or "")
        ws.row_dimensions[row_index].height = row_height_for_movement(movement)

    ws.freeze_panes = f"A{header_row + 1}"
    ws.sheet_view.showGridLines = False
    ws.page_setup.orientation = "portrait"
    ws.page_setup.paperSize = ws.PAPERSIZE_A4
    ws.page_setup.fitToWidth = 1
    ws.page_setup.fitToHeight = 0
    ws.sheet_properties.pageSetUpPr.fitToPage = True
    ws.print_area = f"A{header_row}:{last_col_letter}{last_row}"
    ws.page_margins.left = 0.3
    ws.page_margins.right = 0.3
    ws.page_margins.top = 0.5
    ws.page_margins.bottom = 0.5
    ws.page_margins.footer = 0.2
    ws.oddFooter.center.text = "&P/&N"
    ws.evenFooter.center.text = "&P/&N"
    ws.firstFooter.center.text = "&P/&N"
    ws.sheet_view.zoomScale = 90

    wb.save(output_path)


def export_workbook_to_pdf(excel_path: Path, pdf_path: Path) -> None:
    try:
        import pythoncom
        import win32com.client
        import win32process
    except ImportError as exc:
        raise RuntimeError("La génération PDF nécessite pywin32 et Microsoft Excel.") from exc

    excel_path = excel_path.resolve()
    pdf_path = pdf_path.resolve()
    pdf_path.parent.mkdir(parents=True, exist_ok=True)

    excel = None
    workbook = None
    worksheets = None
    worksheet = None
    page_setup = None
    excel_pid = None
    pythoncom.CoInitialize()
    try:
        excel = win32com.client.DispatchEx("Excel.Application")
        try:
            _, excel_pid = win32process.GetWindowThreadProcessId(excel.Hwnd)
        except Exception:
            excel_pid = None
        excel.Visible = False
        excel.DisplayAlerts = False
        workbook = excel.Workbooks.Open(str(excel_path))

        worksheets = workbook.Worksheets
        for worksheet_index in range(1, worksheets.Count + 1):
            worksheet = worksheets.Item(worksheet_index)
            page_setup = worksheet.PageSetup
            page_setup.Orientation = 1
            page_setup.Zoom = False
            page_setup.FitToPagesWide = 1
            page_setup.FitToPagesTall = False
            page_setup.CenterFooter = "&P/&N"
            page_setup = None
            worksheet = None

        workbook.ExportAsFixedFormat(0, str(pdf_path))
    except Exception as exc:
        raise RuntimeError(f"Impossible de générer le PDF avec Excel : {exc}") from exc
    finally:
        if workbook is not None:
            try:
                workbook.Close(False)
            except Exception:
                pass
        if excel is not None:
            try:
                excel.Quit()
            except Exception:
                pass
        page_setup = None
        worksheet = None
        worksheets = None
        workbook = None
        excel = None
        gc.collect()
        pythoncom.CoFreeUnusedLibraries()
        pythoncom.CoUninitialize()
        if excel_pid is not None:
            time.sleep(0.5)
            subprocess.run(
                ["taskkill", "/PID", str(excel_pid), "/T", "/F"],
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL,
                check=False,
                creationflags=getattr(subprocess, "CREATE_NO_WINDOW", 0),
            )

    if not pdf_path.exists() or pdf_path.stat().st_size == 0:
        raise RuntimeError(f"Le PDF n'a pas été créé : {pdf_path}")


def process_file(
    csv_path: Path,
    treatment: Treatment,
    formats: OutputFormats = EXCEL_ONLY,
    root: Tk | None = None,
) -> OutputInfo:
    if not formats.has_any:
        raise ValueError("Sélectionnez au moins un format.")
    if csv_path.suffix.lower() != ".csv":
        raise ValueError(f"Le fichier n'est pas un CSV : {csv_path.name}")
    if not csv_path.exists():
        raise FileNotFoundError(str(csv_path))

    client, year = derive_client_year(csv_path, root)
    rows = read_rows(csv_path, treatment)
    total_key = "Paiements" if treatment.key == "payments" else "Factures"
    total = sum(float(row.get(total_key) or 0) for row in rows)
    if not rows:
        return OutputInfo(csv_path, None, None, 0, treatment.amount_label, total, treatment.label)

    output_stem = f"{client} - {treatment.output_label} {year}"
    excel_path, pdf_path = unique_output_paths(csv_path.parent, output_stem, formats)

    if formats.excel:
        write_workbook(rows, excel_path, treatment)
        if formats.pdf:
            export_workbook_to_pdf(excel_path, pdf_path)
    elif formats.pdf:
        with tempfile.TemporaryDirectory() as temp_dir:
            temp_excel_path = Path(temp_dir) / f"{output_stem}.xlsx"
            write_workbook(rows, temp_excel_path, treatment)
            export_workbook_to_pdf(temp_excel_path, pdf_path)

    return OutputInfo(csv_path, excel_path, pdf_path, len(rows), treatment.amount_label, total, treatment.label)


def process_many(
    paths: list[Path],
    treatments: tuple[Treatment, ...],
    formats: OutputFormats = EXCEL_ONLY,
    root: Tk | None = None,
) -> list[OutputInfo]:
    if not paths:
        return []
    outputs = []
    errors = []
    for path in paths:
        for treatment in treatments:
            try:
                outputs.append(process_file(path, treatment, formats, root))
            except Exception as exc:
                errors.append(f"{path.name} ({treatment.label}) : {exc}")
    if errors:
        raise RuntimeError("\n".join(errors))
    return outputs


def success_text(outputs: list[OutputInfo]) -> str:
    created = [info for info in outputs if info.excel_output is not None or info.pdf_output is not None]
    skipped = [info for info in outputs if info.excel_output is None and info.pdf_output is None]
    created_file_count = sum(
        int(info.excel_output is not None) + int(info.pdf_output is not None)
        for info in created
    )
    lines = []
    if created:
        lines.append("Fichier généré :" if created_file_count == 1 else "Fichiers générés :")
        for info in created:
            lines.append(f"- {info.treatment_label} : {info.row_count} lignes, {info.amount_label} : {format_amount(info.total_amount)} €")
            if info.excel_output is not None:
                lines.append(f"  Excel : {info.excel_output}")
            if info.pdf_output is not None:
                lines.append(f"  PDF : {info.pdf_output}")
    if skipped:
        if lines:
            lines.append("")
        lines.append("Aucun fichier créé pour :")
        for info in skipped:
            lines.append(f"- {info.treatment_label} : aucune ligne trouvée")
    if not lines:
        lines.append("Aucun fichier généré.")
    return "\n".join(lines)


class App:
    def __init__(self) -> None:
        root_cls = TkinterDnD.Tk if TkinterDnD else Tk
        self.root = root_cls()
        self.root.title("Mise en forme - Grand livre")
        icon_path = resource_path("logo_compta_premium.ico")
        if icon_path.exists():
            try:
                self.root.iconbitmap(str(icon_path))
            except Exception:
                pass
        self.root.geometry("920x700")
        self.root.minsize(820, 660)
        self.root.configure(bg="#F5F7FA")
        saved_formats = load_saved_formats(EXCEL_ONLY, EXCEL_ONLY)
        self.excel_format_var = tk.BooleanVar(value=saved_formats.excel)
        self.pdf_format_var = tk.BooleanVar(value=saved_formats.pdf)
        self.status_var = tk.StringVar(value="Déposez un CSV pour lancer un traitement.")
        self.build_ui()

    def build_ui(self) -> None:
        title = tk.Label(
            self.root,
            text="Mise en forme du grand livre",
            font=("Segoe UI", 18, "bold"),
            bg="#F5F7FA",
            fg="#1F2937",
        )
        title.pack(pady=(22, 6))

        subtitle = tk.Label(
            self.root,
            text="Déposez un export CSV dans la zone correspondant au traitement souhaité.",
            font=("Segoe UI", 10),
            bg="#F5F7FA",
            fg="#4B5563",
        )
        subtitle.pack(pady=(0, 18))

        format_frame = tk.Frame(self.root, bg="#F5F7FA")
        format_frame.pack(pady=(0, 16))
        tk.Label(
            format_frame,
            text="Format de sortie",
            font=("Segoe UI", 10, "bold"),
            bg="#F5F7FA",
            fg="#334155",
        ).pack(side="left", padx=(0, 14))
        for text, variable in (("Excel", self.excel_format_var), ("PDF", self.pdf_format_var)):
            tk.Checkbutton(
                format_frame,
                text=text,
                variable=variable,
                command=self.save_selected_formats,
                font=("Segoe UI", 10),
                bg="#F5F7FA",
                fg="#1F2937",
                activebackground="#F5F7FA",
                activeforeground="#1F2937",
                selectcolor="#FFFFFF",
                cursor="hand2",
            ).pack(side="left", padx=8)

        row_frame = tk.Frame(self.root, bg="#F5F7FA")
        row_frame.pack(fill="x", padx=34)
        row_frame.columnconfigure(0, weight=1)
        row_frame.columnconfigure(1, weight=1)

        self.create_drop_box(
            row_frame,
            title="Lister les paiements\nsans facture",
            treatments=(PAYMENTS_WITHOUT_INVOICE,),
            column=0,
            accent="#1F4E78",
        )
        self.create_drop_box(
            row_frame,
            title="Lister les factures\nsans paiements",
            treatments=(INVOICES_WITHOUT_PAYMENT,),
            column=1,
            accent="#8A6A13",
        )

        bottom_frame = tk.Frame(self.root, bg="#F5F7FA")
        bottom_frame.pack(fill="x", padx=210, pady=(20, 0))
        bottom_frame.columnconfigure(0, weight=1)
        self.create_drop_box(
            bottom_frame,
            title="Générer les 2 fichiers",
            treatments=ALL_TREATMENTS,
            column=0,
            accent="#334155",
            compact=True,
        )

        note = "Le dépôt direct d'un CSV sur l'icône du .exe génère les paiements sans facture en Excel + PDF."
        if not DND_FILES:
            note = "Le glisser-déposer dans la fenêtre est indisponible ; utilisez les boutons ou glissez sur le .exe."
        tk.Label(self.root, text=note, font=("Segoe UI", 9), bg="#F5F7FA", fg="#4B5563").pack(pady=(18, 0))

        self.status_frame = tk.Frame(
            self.root,
            bg="#FFFFFF",
            highlightbackground="#CBD5E1",
            highlightthickness=1,
            bd=0,
            height=105,
        )
        self.status_frame.pack(fill="x", padx=46, pady=(14, 0))
        self.status_frame.pack_propagate(False)

        self.status_title = tk.Label(
            self.status_frame,
            text="Résultat",
            font=("Segoe UI", 10, "bold"),
            bg="#FFFFFF",
            fg="#334155",
            anchor="w",
        )
        self.status_title.pack(fill="x", padx=14, pady=(10, 2))

        self.status_label = tk.Label(
            self.status_frame,
            textvariable=self.status_var,
            font=("Segoe UI", 9),
            bg="#FFFFFF",
            fg="#475569",
            justify="left",
            anchor="nw",
            wraplength=800,
        )
        self.status_label.pack(fill="both", expand=True, padx=14, pady=(0, 10))

    def create_drop_box(
        self,
        parent: tk.Frame,
        title: str,
        treatments: tuple[Treatment, ...],
        column: int,
        accent: str,
        compact: bool = False,
    ) -> None:
        frame = tk.Frame(
            parent,
            bg="#FFFFFF",
            highlightbackground=accent,
            highlightthickness=2,
            bd=0,
            height=125 if compact else 180,
            cursor="hand2",
        )
        frame.grid(row=0, column=column, padx=12, sticky="nsew")
        frame.pack_propagate(False)

        label = tk.Label(
            frame,
            text=title,
            font=("Segoe UI", 17 if compact else 18, "bold"),
            bg="#FFFFFF",
            fg=accent,
            justify="center",
            cursor="hand2",
        )
        label.pack(pady=(26 if compact else 42, 16))

        button = tk.Button(
            frame,
            text="Choisir un CSV",
            command=lambda: self.choose_file(treatments),
            font=("Segoe UI", 10),
            bg=accent,
            fg="#FFFFFF",
            activebackground=accent,
            activeforeground="#FFFFFF",
            relief="flat",
            padx=14,
            pady=6,
            cursor="hand2",
        )
        button.pack()

        for widget in (frame, label):
            widget.bind("<Button-1>", lambda _event, selected=treatments: self.choose_file(selected))

        if DND_FILES:
            for widget in (frame, label):
                widget.drop_target_register(DND_FILES)
                widget.dnd_bind("<<Drop>>", lambda event, selected=treatments: self.on_drop(event, selected))

    def on_drop(self, event, treatments: tuple[Treatment, ...]) -> None:
        self.handle_paths([Path(item) for item in self.root.tk.splitlist(event.data)], treatments)

    def choose_file(self, treatments: tuple[Treatment, ...]) -> None:
        files = filedialog.askopenfilenames(
            parent=self.root,
            title="Choisir un ou plusieurs CSV",
            filetypes=[("Fichiers CSV", "*.csv"), ("Tous les fichiers", "*.*")],
        )
        self.handle_paths([Path(file) for file in files], treatments)

    def set_status(self, text: str, kind: str = "info") -> None:
        colors = {
            "info": ("#FFFFFF", "#CBD5E1", "#334155", "#475569"),
            "success": ("#F0FDF4", "#86EFAC", "#166534", "#14532D"),
            "error": ("#FEF2F2", "#FCA5A5", "#991B1B", "#7F1D1D"),
        }
        bg, border, title_fg, text_fg = colors.get(kind, colors["info"])
        self.status_frame.configure(bg=bg, highlightbackground=border)
        self.status_title.configure(bg=bg, fg=title_fg)
        self.status_label.configure(bg=bg, fg=text_fg)
        self.status_var.set(text)

    def selected_formats(self) -> OutputFormats:
        return OutputFormats(
            excel=bool(self.excel_format_var.get()),
            pdf=bool(self.pdf_format_var.get()),
        )

    def save_selected_formats(self) -> None:
        try:
            save_formats(self.selected_formats())
        except Exception as exc:
            self.set_status(f"Erreur lors de la sauvegarde du format :\n{exc}", "error")

    def handle_paths(self, paths: list[Path], treatments: tuple[Treatment, ...]) -> None:
        if not paths:
            return
        formats = self.selected_formats()
        if not formats.has_any:
            self.set_status("Sélectionnez au moins un format.", "error")
            return
        try:
            outputs = process_many(paths, treatments, formats, self.root)
            created = any(info.excel_output is not None or info.pdf_output is not None for info in outputs)
            self.set_status(success_text(outputs), "success" if created else "info")
        except Exception as exc:
            write_error_log(str(exc))
            self.set_status(f"Erreur :\n{exc}", "error")

    def run(self) -> None:
        self.root.mainloop()


def main() -> int:
    no_dialog = "--no-dialog" in sys.argv
    args = [Path(arg) for arg in sys.argv[1:] if arg != "--no-dialog"]
    if args:
        root = None if no_dialog else Tk()
        if root is not None:
            root.withdraw()
        try:
            formats = load_saved_formats(EXCEL_AND_PDF, EXCEL_ONLY)
            outputs = process_many(args, (PAYMENTS_WITHOUT_INVOICE,), formats, root)
            if not no_dialog:
                return 0
            if sys.stdout is not None:
                print(success_text(outputs))
            return 0
        except Exception as exc:
            write_error_log(str(exc))
            if no_dialog:
                if sys.stderr is not None:
                    print(str(exc), file=sys.stderr)
            else:
                messagebox.showerror("Erreur", str(exc), parent=root)
            return 1
        finally:
            if root is not None:
                root.destroy()

    App().run()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
