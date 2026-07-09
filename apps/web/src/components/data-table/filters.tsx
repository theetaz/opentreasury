import { useEffect, useState } from "react";
import { format } from "date-fns";
import { CalendarRange, X } from "lucide-react";
import type { DateRange } from "react-day-picker";
import { Button } from "@/components/ui/button";
import { Calendar } from "@/components/ui/calendar";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from "@/components/ui/select";

/** Wrapping toolbar for typed filter controls; stacks on small screens. */
export function FilterBar({ children }: { children: React.ReactNode }) {
  return (
    <Card>
      <CardContent className="grid grid-cols-1 items-end gap-3 sm:grid-cols-2 lg:flex lg:flex-wrap">
        {children}
      </CardContent>
    </Card>
  );
}

type Option = { value: string; label: string };

const CLEAR = "__all__";

/** Dropdown filter for relationships and enumerations — never free text. */
export function SelectFilter({
  label,
  value,
  options,
  onChange,
  placeholder = "All"
}: {
  label: string;
  value: string | undefined;
  options: Option[];
  onChange: (value: string | undefined) => void;
  placeholder?: string;
}) {
  return (
    <div className="grid gap-1.5">
      <Label>{label}</Label>
      <Select
        value={value ?? CLEAR}
        onValueChange={(next) => onChange(next === CLEAR ? undefined : next)}
      >
        <SelectTrigger className="w-full lg:w-44" aria-label={label}>
          <SelectValue placeholder={placeholder} />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value={CLEAR}>{placeholder}</SelectItem>
          {options.map((option) => (
            <SelectItem key={option.value} value={option.value}>
              {option.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}

/** Inclusive date-range filter backed by a range calendar. */
export function DateRangeFilter({
  label = "Date range",
  dateFrom,
  dateTo,
  onChange
}: {
  label?: string;
  dateFrom: string | undefined;
  dateTo: string | undefined;
  onChange: (dateFrom: string | undefined, dateTo: string | undefined) => void;
}) {
  const selected: DateRange | undefined = dateFrom
    ? { from: new Date(dateFrom), to: dateTo ? new Date(dateTo) : undefined }
    : undefined;

  const display =
    dateFrom && dateTo ? `${dateFrom} → ${dateTo}` : dateFrom ? `From ${dateFrom}` : "Any time";

  return (
    <div className="grid gap-1.5">
      <Label>{label}</Label>
      <div className="flex items-center gap-1">
        <Popover>
          <PopoverTrigger asChild>
            <Button variant="outline" className="w-full justify-start font-normal lg:w-60" aria-label={label}>
              <CalendarRange aria-hidden className="text-muted-foreground" />
              <span className="figures-tabular">{display}</span>
            </Button>
          </PopoverTrigger>
          <PopoverContent className="w-auto p-0" align="start">
            <Calendar
              mode="range"
              numberOfMonths={2}
              selected={selected}
              onSelect={(range) => {
                onChange(
                  range?.from ? format(range.from, "yyyy-MM-dd") : undefined,
                  range?.to ? format(range.to, "yyyy-MM-dd") : undefined
                );
              }}
            />
          </PopoverContent>
        </Popover>
        {dateFrom ? (
          <Button
            variant="ghost"
            size="icon"
            aria-label="Clear date range"
            onClick={() => onChange(undefined, undefined)}
          >
            <X aria-hidden />
          </Button>
        ) : null}
      </div>
    </div>
  );
}

type AmountOperator = "gte" | "lte" | "eq" | "between";

function deriveOperator(gte: string | undefined, lte: string | undefined): AmountOperator {
  if (gte && lte) return gte === lte ? "eq" : "between";
  if (lte) return "lte";
  return "gte";
}

/**
 * Amount filter with comparison operators. Values are entered in major units
 * and travel as minor units (amountGte/amountLte) in the URL and the API.
 */
export function AmountFilter({
  amountGte,
  amountLte,
  onChange
}: {
  amountGte: string | undefined;
  amountLte: string | undefined;
  onChange: (amountGte: string | undefined, amountLte: string | undefined) => void;
}) {
  const [operator, setOperator] = useState<AmountOperator>(() => deriveOperator(amountGte, amountLte));
  const [primary, setPrimary] = useState(() => toMajor(operator === "lte" ? amountLte : amountGte));
  const [secondary, setSecondary] = useState(() => toMajor(amountLte));

  // Re-sync when the URL changes externally (back button, shared link).
  useEffect(() => {
    const derived = deriveOperator(amountGte, amountLte);
    setOperator(derived);
    setPrimary(toMajor(derived === "lte" ? amountLte : amountGte));
    setSecondary(toMajor(amountLte));
  }, [amountGte, amountLte]);

  function apply() {
    const value = toMinor(primary);
    const upper = toMinor(secondary);

    switch (operator) {
      case "gte":
        onChange(value, undefined);
        break;
      case "lte":
        onChange(undefined, value);
        break;
      case "eq":
        onChange(value, value);
        break;
      case "between":
        onChange(value, upper);
        break;
    }
  }

  return (
    <div className="grid gap-1.5">
      <Label htmlFor="amount-filter-value">Amount</Label>
      <div className="flex items-center gap-1.5">
        <Select value={operator} onValueChange={(next) => setOperator(next as AmountOperator)}>
          <SelectTrigger className="w-24" aria-label="Amount operator">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="gte">≥</SelectItem>
            <SelectItem value="lte">≤</SelectItem>
            <SelectItem value="eq">=</SelectItem>
            <SelectItem value="between">between</SelectItem>
          </SelectContent>
        </Select>
        <Input
          id="amount-filter-value"
          type="number"
          min="0"
          step="0.01"
          inputMode="decimal"
          placeholder="0.00"
          className="w-28 font-mono"
          value={primary}
          onChange={(event) => setPrimary(event.target.value)}
          onKeyDown={(event) => event.key === "Enter" && apply()}
        />
        {operator === "between" ? (
          <Input
            type="number"
            min="0"
            step="0.01"
            inputMode="decimal"
            placeholder="0.00"
            aria-label="Amount upper bound"
            className="w-28 font-mono"
            value={secondary}
            onChange={(event) => setSecondary(event.target.value)}
            onKeyDown={(event) => event.key === "Enter" && apply()}
          />
        ) : null}
        <Button variant="secondary" onClick={apply}>
          Apply
        </Button>
        {(amountGte ?? amountLte) ? (
          <Button
            variant="ghost"
            size="icon"
            aria-label="Clear amount filter"
            onClick={() => onChange(undefined, undefined)}
          >
            <X aria-hidden />
          </Button>
        ) : null}
      </div>
    </div>
  );
}

function toMajor(minor: string | undefined): string {
  if (!minor) return "";
  const value = Number(minor);
  return Number.isFinite(value) ? String(value / 100) : "";
}

function toMinor(major: string): string | undefined {
  if (major.trim() === "") return undefined;
  const value = Number(major);
  if (!Number.isFinite(value) || value < 0) return undefined;
  return String(Math.round(value * 100));
}
