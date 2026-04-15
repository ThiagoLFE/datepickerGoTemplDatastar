/* See end of file for copyright information */

let weekdayLabels = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];

export function useWeekdayLabels(labels: string[]) {
    assert(
        labels.length == 7,
        `Invalid number of week days: ${labels.length}. Should be 7`,
    );
    weekdayLabels = labels;
}

function assert(pred: boolean, msg: string) {
    if (!pred) {
        throw new Error(`assertion failed: ${msg}`);
    }
}

function pad2(n: number): string {
    return String(n).padStart(2, "0");
}

export class DateOnly {
    private _date: Date;

    constructor(year: number, month: number, day: number) {
        this._date = new Date(Date.UTC(year, month, day));
    }

    static fromDate(d: Date): DateOnly {
        return new DateOnly(
            d.getUTCFullYear(),
            d.getUTCMonth(),
            d.getUTCDate(),
        );
    }

    get year(): number {
        return this._date.getUTCFullYear();
    }
    get month(): number {
        return this._date.getUTCMonth();
    }
    get day(): number {
        return this._date.getUTCDate();
    }

    before(other: DateOnly): boolean {
        return this._date.getTime() < other._date.getTime();
    }

    after(other: DateOnly): boolean {
        return this._date.getTime() > other._date.getTime();
    }

    equals(other: DateOnly): boolean {
        return this._date.getTime() === other._date.getTime();
    }

    previousMonth(): DateOnly {
        return new DateOnly(this.year, this.month - 1, 1);
    }

    nextMonth(): DateOnly {
        return new DateOnly(this.year, this.month + 1, 1);
    }

    daysInMonth(): number {
        return new Date(Date.UTC(this.year, this.month + 1, 0)).getUTCDate();
    }

    dayOffset(): number {
        return new Date(Date.UTC(this.year, this.month, 1)).getUTCDay();
    }
}

// ======================= Essas funções provavelmente vão para um arquivo separado ===============

export function toBrazileanDateFormat(date: DateOnly): string {
    return `${date.day}/${pad2(date.month + 1)}/${pad2(date.year)}`;
}

export function brazileanToDateOnly(brazileanDate: string): DateOnly | null {
    const parts = brazileanDate.split("/");

    if (parts.length != 3) return null;

    const day = Number.parseInt(parts[0]);
    const month = Number.parseInt(parts[1]) - 1;
    const year = Number.parseInt(parts[2]);

    if (year < 1000) return null; // certificando os 4 digitos;

    return new DateOnly(year, month, day);
}

type DatepickerConfig = {
    startDate?: DateOnly;
    ranged?: boolean;
    endDate?: DateOnly;
    minDate?: DateOnly;
    maxDate?: DateOnly;
    toDisplay?: (date: DateOnly) => string;
    fromDisplay?: (displayDate: string) => DateOnly;
    onSelect?:
        | ((self: Datepicker, date?: DateOnly) => any)
        | ((self: Datepicker, begin?: DateOnly, end?: DateOnly) => any);
};

export class Datepicker {
    year: number;
    month: number;

    startDateDisplay: string;
    startDateValue: DateOnly | null;

    endDateDisplay: string;
    endDateValue: DateOnly | null;

    onSelect?: (self: Datepicker, begin: DateOnly, end?: DateOnly) => any;
    toDisplay: (date: DateOnly) => string;
    fromDisplay: (displayDate: string) => DateOnly | null;

    ranged: boolean;
    double: boolean;

    minDate?: DateOnly;
    maxDate?: DateOnly;

    element: HTMLDivElement;

    constructor(config: DatepickerConfig = {}) {
        const now = new Date();

        this.year = config.startDate?.year ?? now.getUTCFullYear();
        this.month = config.startDate?.month ?? now.getUTCMonth();

        this.fromDisplay = config.fromDisplay ?? Datepicker.ISO8601ToDateOnly;
        this.toDisplay = config.toDisplay ?? Datepicker.toISO8601;

        this.startDateValue = config.startDate ?? null;
        this.startDateDisplay = this.startDateValue
            ? this.toDisplay(this.startDateValue)
            : "";

        this.endDateValue = config.endDate ?? null;
        this.endDateDisplay = this.endDateValue
            ? this.toDisplay(this.endDateValue)
            : "";

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
            if (this.minDate && d.before(this.minDate)) d = this.minDate;
            if (this.maxDate && d.after(this.maxDate)) d = this.maxDate;
            this.startDateValue = d;
        } else {
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
            if (this.minDate && d.before(this.minDate)) d = this.minDate;
            if (this.maxDate && d.after(this.maxDate)) d = this.maxDate;
            this.endDateValue = d;
        } else {
            this.endDateValue = null;
        }
        this._render();
    }

    _handleClick(date: DateOnly): void {
        if (this.ranged) {
            if (
                !this.startDateValue ||
                (this.startDateValue && this.endDateValue)
            ) {
                this.startDateValue = date;
                this.endDateValue = null;
            } else {
                if (date.before(this.startDateValue)) {
                    this.endDateValue = this.startDateValue;
                    this.startDateValue = date;
                } else {
                    this.endDateValue = date;
                }
                if (this.onSelect) {
                    this.onSelect(this, this.startDateValue, this.endDateValue);
                }
            }
        } else {
            this.startDateValue = date;
            this.endDateValue = null;
            if (this.onSelect) this.onSelect(this, this.startDateValue);
        }
        this._render();
    }

    static toISO8601(date: DateOnly): string {
        return `${date.year}-${pad2(date.month + 1)}-${pad2(date.day)}`;
    }

    static ISO8601ToDateOnly(dateISO8601: string): DateOnly | null {
        const parts = dateISO8601.split("-");

        if (parts.length != 3) return null;

        const year = Number.parseInt(parts[0]);
        const month = Number.parseInt(parts[1]) - 1;
        const day = Number.parseInt(parts[2]);

        if (year < 1000) return null; // certificando os 4 digitos;

        return new DateOnly(year, month, day);
    }

    _prevMonth(): void {
        const prev = new DateOnly(this.year, this.month, 1).previousMonth();
        this.year = prev.year;
        this.month = prev.month;
        this._render();
    }

    _nextMonth(): void {
        const next = new DateOnly(this.year, this.month, 1).nextMonth();
        this.year = next.year;
        this.month = next.month;
        this._render();
    }

    _renderGrid(year: number, month: number): HTMLDivElement {
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
            let value: DateOnly;

            btn.className = "datepicker-cell";

            let inCurrentMonth = true;

            if (day > numDays) {
                btn.setAttribute("disabled", "true");
                inCurrentMonth = false;
                day = i - numDays + 1 - dayOffset;
                value = new DateOnly(next.year, next.month, day);
            } else if (day <= 0) {
                btn.setAttribute("disabled", "true");
                inCurrentMonth = false;
                day = prev.daysInMonth() - dayOffset + i + 1;
                value = new DateOnly(prev.year, prev.month, day);
            } else {
                value = new DateOnly(year, month, day);
            }

            if (
                (this.minDate && value.before(this.minDate)) ||
                (this.maxDate && value.after(this.maxDate))
            ) {
                btn.setAttribute("disabled", "true");
            }

            if (inCurrentMonth && this.startDate && this.endDate) {
                if (
                    value.equals(this.startDate) ||
                    value.equals(this.endDate)
                ) {
                    btn.classList.add("datepicker-selected");
                } else if (
                    value.after(this.startDate) &&
                    value.before(this.endDate)
                ) {
                    btn.classList.add("datepicker-range");
                }
            } else if (
                inCurrentMonth &&
                this.startDate &&
                value.equals(this.startDate)
            ) {
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

    _render(): void {
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

        const fmtMonth = (y: number, m: number): string =>
            new Date(Date.UTC(y, m, 1)).toLocaleString("default", {
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
        } else {
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
        } else {
            this.element.appendChild(this._renderGrid(this.year, this.month));
        }
    }

    mount(container: string | HTMLElement): this {
        const el: HTMLElement =
            typeof container === "string"
                ? document.querySelector<HTMLElement>(container)!
                : container;
        el.appendChild(this.element);
        return this;
    }
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
