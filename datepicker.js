/* See end of file for copyright information */
let weekdayLabels = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];
export function useWeekdayLabels(labels) {
    assert(labels.length == 7, `Invalid number of week days: ${labels.length}. Should be 7`);
    weekdayLabels = labels;
}
function assert(pred, msg) {
    if (!pred) {
        throw new Error(`assertion failed: ${msg}`);
    }
}
function pad2(n) {
    return String(n).padStart(2, "0");
}
export class DateOnly {
    _date;
    constructor(year, month, day) {
        this._date = new Date(Date.UTC(year, month, day));
    }
    static now() {
        return DateOnly.fromDate(new Date());
    }
    static fromDate(d) {
        return new DateOnly(d.getUTCFullYear(), d.getUTCMonth(), d.getUTCDate());
    }
    get year() {
        return this._date.getUTCFullYear();
    }
    get month() {
        return this._date.getUTCMonth();
    }
    get day() {
        return this._date.getUTCDate();
    }
    before(other) {
        return this._date.getTime() < other._date.getTime();
    }
    after(other) {
        return this._date.getTime() > other._date.getTime();
    }
    equals(other) {
        return this._date.getTime() === other._date.getTime();
    }
    previousMonth() {
        return new DateOnly(this.year, this.month - 1, 1);
    }
    nextMonth() {
        return new DateOnly(this.year, this.month + 1, 1);
    }
    daysInMonth() {
        return new Date(Date.UTC(this.year, this.month + 1, 0)).getUTCDate();
    }
    dayOffset() {
        return new Date(Date.UTC(this.year, this.month, 1)).getUTCDay();
    }
}
// ======================= Essas funções provavelmente vão para um arquivo separado ===============
export function toBrazileanDateFormat(date) {
    return `${date.day}/${pad2(date.month + 1)}/${pad2(date.year)}`;
}
export function brazileanToDateOnly(brazileanDate) {
    const parts = brazileanDate.split("/");
    if (parts.length != 3)
        return null;
    const day = Number.parseInt(parts[0]);
    const month = Number.parseInt(parts[1]) - 1;
    const year = Number.parseInt(parts[2]);
    if (year < 1000)
        return null; // certificando os 4 digitos;
    return new DateOnly(year, month, day);
}
export class Datepicker {
    year;
    month;
    startDateValue;
    endDateValue;
    onSelect;
    toDisplay;
    fromDisplay;
    ranged;
    double;
    minDate;
    maxDate;
    element;
    constructor(config = {}) {
        const now = new Date();
        this.year = config.startDate?.year ?? now.getUTCFullYear();
        this.month = config.startDate?.month ?? now.getUTCMonth();
        this.fromDisplay = config.fromDisplay ?? Datepicker.ISO8601ToDateOnly;
        this.toDisplay = config.toDisplay ?? Datepicker.toISO8601;
        this.startDateValue = config.startDate ?? null;
        this.endDateValue = config.endDate ?? null;
        this.ranged = config.ranged ?? false;
        this.double = config.ranged ?? false;
        this.minDate = config.minDate ?? undefined;
        this.maxDate = config.maxDate ?? undefined;
        this.element = document.createElement("div");
        this.element.className = this.double
            ? "datepicker datepicker-double"
            : "datepicker";
        this.onSelect = config.onSelect ?? undefined;
        this._render();
    }
    get startDate() {
        return this.startDateValue;
    }
    set startDate(v) {
        if (v) {
            let d = v;
            if (this.minDate && d.before(this.minDate))
                d = this.minDate;
            if (this.maxDate && d.after(this.maxDate))
                d = this.maxDate;
            this.startDateValue = d;
        }
        else {
            this.startDateValue = null;
        }
        this._render();
    }
    get endDate() {
        return this.endDateValue;
    }
    set endDate(v) {
        if (v) {
            let d = v;
            if (this.minDate && d.before(this.minDate))
                d = this.minDate;
            if (this.maxDate && d.after(this.maxDate))
                d = this.maxDate;
            this.endDateValue = d;
        }
        else {
            this.endDateValue = null;
        }
        this._render();
    }
    get startDateDisplay() {
        if (!this.startDateValue)
            return "";
        return this.toDisplay(this.startDateValue);
    }
    get endDateDisplay() {
        if (!this.startDateValue)
            return "";
        return this.toDisplay(this.endDateValue);
    }
    _handleClick(date) {
        if (this.ranged) {
            if (!this.startDateValue ||
                (this.startDateValue && this.endDateValue)) {
                this.startDateValue = date;
                this.endDateValue = null;
            }
            else {
                if (date.before(this.startDateValue)) {
                    this.endDateValue = this.startDateValue;
                    this.startDateValue = date;
                }
                else {
                    this.endDateValue = date;
                }
                if (this.onSelect) {
                    this.onSelect(this, this.startDateValue, this.endDateValue);
                }
            }
        }
        else {
            this.startDateValue = date;
            this.endDateValue = null;
            if (this.onSelect)
                this.onSelect(this, this.startDateValue);
        }
        this._render();
    }
    static toISO8601(date) {
        return `${date.year}-${pad2(date.month + 1)}-${pad2(date.day)}`;
    }
    static ISO8601ToDateOnly(dateISO8601) {
        const parts = dateISO8601.split("-");
        if (parts.length != 3)
            return null;
        const year = Number.parseInt(parts[0]);
        const month = Number.parseInt(parts[1]) - 1;
        const day = Number.parseInt(parts[2]);
        if (year < 1000)
            return null; // certificando os 4 digitos;
        return new DateOnly(year, month, day);
    }
    _prevMonth() {
        const prev = new DateOnly(this.year, this.month, 1).previousMonth();
        this.year = prev.year;
        this.month = prev.month;
        this._render();
    }
    _nextMonth() {
        const next = new DateOnly(this.year, this.month, 1).nextMonth();
        this.year = next.year;
        this.month = next.month;
        this._render();
    }
    _renderGrid(year, month) {
        const grid = document.createElement("div");
        grid.className = "datepicker-grid";
        for (let i = 0; i < weekdayLabels.length; i++) {
            let lbl = document.createElement("span");
            lbl.innerText = weekdayLabels[i];
            grid.appendChild(lbl);
        }
        const ref = new DateOnly(year, month, 1);
        const dayOffset = ref.dayOffset();
        const numDays = ref.daysInMonth();
        const prev = ref.previousMonth();
        const next = ref.nextMonth();
        for (let i = 0; i < 42; i++) {
            let btn = document.createElement("button");
            let day = i + 1 - dayOffset;
            let value;
            btn.className = "datepicker-cell";
            let inCurrentMonth = true;
            if (day > numDays) {
                btn.setAttribute("disabled", "true");
                inCurrentMonth = false;
                day = i - numDays + 1 - dayOffset;
                value = new DateOnly(next.year, next.month, day);
            }
            else if (day <= 0) {
                btn.setAttribute("disabled", "true");
                inCurrentMonth = false;
                day = prev.daysInMonth() - dayOffset + i + 1;
                value = new DateOnly(prev.year, prev.month, day);
            }
            else {
                value = new DateOnly(year, month, day);
            }
            if ((this.minDate && value.before(this.minDate)) ||
                (this.maxDate && value.after(this.maxDate))) {
                btn.setAttribute("disabled", "true");
            }
            if (inCurrentMonth && this.startDate && this.endDate) {
                if (value.equals(this.startDate) ||
                    value.equals(this.endDate)) {
                    btn.classList.add("datepicker-selected");
                }
                else if (value.after(this.startDate) &&
                    value.before(this.endDate)) {
                    btn.classList.add("datepicker-range");
                }
            }
            else if (inCurrentMonth &&
                this.startDate &&
                value.equals(this.startDate)) {
                btn.classList.add("datepicker-selected");
            }
            btn.addEventListener("click", (ev) => {
                ev.preventDefault();
                this._handleClick(value);
            });
            btn.innerText = `${day}`;
            grid.appendChild(btn);
        }
        return grid;
    }
    _render() {
        this.element.innerHTML = "";
        const nextRef = new DateOnly(this.year, this.month, 1).nextMonth();
        const nextYear = nextRef.year;
        const nextMonth = nextRef.month;
        // Header
        const header = document.createElement("div");
        header.className = "datepicker-header";
        // Header > Left side
        const prevBtn = document.createElement("button");
        prevBtn.className = "datepicker-header-nav";
        prevBtn.innerText = "<";
        prevBtn.addEventListener("click", (ev) => {
            ev.preventDefault();
            this._prevMonth();
        });
        // Header > Right side
        const nextBtn = document.createElement("button");
        nextBtn.className = "datepicker-header-nav";
        nextBtn.innerText = ">";
        nextBtn.addEventListener("click", (ev) => {
            ev.preventDefault();
            this._nextMonth();
        });
        const fmtMonth = (y, m) => new Date(Date.UTC(y, m, 1)).toLocaleString("default", {
            month: "long",
            year: "numeric",
            timeZone: "UTC",
        });
        if (this.double) {
            const label1 = document.createElement("span");
            label1.innerText = fmtMonth(this.year, this.month);
            const label2 = document.createElement("span");
            label2.innerText = fmtMonth(nextYear, nextMonth);
            header.appendChild(prevBtn);
            header.appendChild(label1);
            header.appendChild(label2);
            header.appendChild(nextBtn);
        }
        else {
            const label = document.createElement("span");
            label.innerText = fmtMonth(this.year, this.month);
            header.appendChild(prevBtn);
            header.appendChild(label);
            header.appendChild(nextBtn);
        }
        this.element.appendChild(header);
        // Grid
        if (this.double) {
            const months = document.createElement("div");
            months.className = "datepicker-months";
            months.appendChild(this._renderGrid(this.year, this.month));
            months.appendChild(this._renderGrid(nextYear, nextMonth));
            this.element.appendChild(months);
        }
        else {
            this.element.appendChild(this._renderGrid(this.year, this.month));
        }
    }
    mount(container) {
        const el = typeof container === "string"
            ? document.querySelector(container)
            : container;
        el.appendChild(this.element);
        return this;
    }
}
// =========================== InputDate =================================
export class InputDate {
    _inputRaw;
    _inputMask;
    _value;
    inputElement;
    onChange;
    constructor(options) {
        options = options ?? {};
        this.inputElement = target;
        options.initialValue ??= DateOnly.now();
        // Verificando se vai ser iniciado com valor ou nulo;
        const parts = splitBrazileanMask(options.initialValue);
        if (parts.length != 3) {
            this._inputRaw = "";
            this._inputMask = "";
            this._value = null;
        }
        else {
            this._inputRaw = onlyNumbers(limitNumbersLength(options.initialValue));
            this._inputMask = options.initialValue;
            this._value = dateBrazileanToDateOnly(options.initialValue);
        }
    }
    // getters & setters que sincronizam estado
    update(rawInput) {
        this._inputMask = applyBrazileanMask(this._inputRaw);
        this._value = dateBrazileanToDateOnly(this._inputMask);
        if (this.onChange) {
            this.onChange(this, this._value);
        }
    }
    get value() {
        return this._value;
    }
}
function onlyNumbers(brazileanInputMask) {
    return brazileanInputMask.replace(/\D/g, "");
}
function limitNumbersLength(inputRaw) {
    return inputRaw.slice(0, 10);
}
function splitBrazileanMask(brazileanDate) {
    return brazileanDate.split("/");
}
function applyBrazileanMask(value) {
    const digits = limitNumbersLength(onlyNumbers(value));
    if (digits.length <= 2)
        return digits;
    if (digits.length < 5)
        return `${digits.slice(0, 2)}/${digits.slice(2)}`;
    return `${digits.slice(0, 2)}/${digits.slice(2, 4)}/${digits.slice(4, 8)}`;
}
function dateBrazileanToDateOnly(dateBrazilean) {
    let parts = splitBrazileanMask(dateBrazilean);
    if (parts.length != 3) {
        return null;
    }
    const day = Number.parseInt(parts[0]);
    const month = Number.parseInt(parts[1]) - 1;
    const year = Number.parseInt(parts[2]);
    if (year < 1000) {
        return null;
    }
    return new DateOnly(year, month, day);
}
/*
    MIT No Attribution

    Copyright 2026 axe-or

    Permission is hereby granted, free of charge, to any person obtaining a copy of this
    software and associated documentation files (the "Software"), to deal in the Software
    without restriction, including without limitation the rights to use, copy, modify,
    merge, publish, distribute, sublicense, and/or sell copies of the Software, and to
    permit persons to whom the Software is furnished to do so.

    THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED,
    INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A
    PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT
    HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
    OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
    SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/
