import {
    Datepicker,
    toBrazileanDateFormat,
    brazileanToDateOnly,
} from "/datepicker.js";

import { root } from "/datastar.js";

export let datepickers = {};

export function createDatepicker(id, options = {}) {
    const target = document.getElementById(id);
    if (!target) {
        throw new Error(`nao achei ${id}`);
    }

    const picker = new Datepicker({
        startDate: options.startDate,
        ranged: true,
        endDate: options.endDate,
        minDate: options.minDate ?? null,
        maxDate: options.maxDate ?? null,
        fromDisplay: toBrazileanDateFormat, //Chumbado para usarmos no Brasil
        toDisplay: brazileanToDateOnly, //Chumbado para usarmos no Brasil
        onSelect: (self, begin, end) => {
            if (root?.datepickers) {
                root.datepickers[id] = {
                    begin: toBrazileanDateFormat(begin),
                    end: toBrazileanDateFormat(end),
                };
            }

            self.startDateValue = brazileanToDateOnly(
                root.datepickers[id]["begin"],
            );
            self.endDateValue = brazileanToDateOnly(
                root.datepickers[id]["end"],
            );

            if (typeof options.onSelect === "function") {
                options.onSelect(begin, end);
            }
            self._render();
        },
    }).mount(target);

    datepickers[id] = picker;
    return picker;
}

export function applyDateMask(evt) {
    let value = evt.target.value;

    value = filterNumbers(value);
    value = limitDateDigits(value);
    value = dateMaskABNT(value);

    evt.target.value = value;
}

function filterNumbers(value) {
    return value.replace(/\D/g, "");
}

function limitDateDigits(value) {
    return value.slice(0, 8);
}

function dateMaskABNT(value) {
    const digits = limitDateDigits(filterNumbers(value));

    if (digits.length <= 2) return digits;
    if (digits.length < 5) return `${digits.slice(0, 2)}/${digits.slice(2)}`;

    return `${digits.slice(0, 2)}/${digits.slice(2, 4)}/${digits.slice(4, 8)}`;
}
