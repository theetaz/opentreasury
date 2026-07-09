import { useMemo, useState } from "react";
import type { ColumnDef } from "@tanstack/react-table";
import { BookText, Plus, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { DataTable } from "@/components/data-table/data-table";
import { DateRangeFilter, FilterBar, SelectFilter } from "@/components/data-table/filters";
import { useTableUrlState } from "@/components/data-table/use-table-url-state";
import { PageHeader } from "@/components/page-header";
import { StatusBadge } from "@/components/status-badge";
import { VerifyEntry } from "@/components/verify-entry";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from "@/components/ui/select";
import { useInstitutions, useJournalEntries, usePostJournalEntry } from "@/hooks/use-treasury";
import { formatMoney } from "@/lib/money";
import { cn } from "@/lib/utils";
import type { JournalEntry, JournalLine } from "@/api";

function entryDebitTotal(entry: JournalEntry): { currency: string; amount: number } {
  const currency = entry.lines[0]?.currency ?? "USD";
  const amount = entry.lines
    .filter((line) => line.direction === "DEBIT")
    .reduce((sum, line) => sum + line.amountMinor, 0);
  return { currency, amount };
}

const columns: ColumnDef<JournalEntry, unknown>[] = [
  {
    accessorKey: "id",
    header: "Entry",
    cell: ({ row }) => <span className="font-mono text-[13px]">{row.original.id}</span>
  },
  { accessorKey: "institutionId", header: "Institution" },
  {
    accessorKey: "effectiveDate",
    header: "Effective",
    cell: ({ row }) => <span className="figures-tabular">{row.original.effectiveDate}</span>
  },
  { accessorKey: "description", header: "Description" },
  {
    id: "amount",
    header: () => <div className="text-right">Amount</div>,
    cell: ({ row }) => {
      const { currency, amount } = entryDebitTotal(row.original);
      return <div className="text-right figures-tabular">{formatMoney(currency, amount)}</div>;
    }
  },
  {
    id: "lines",
    header: "Lines",
    cell: ({ row }) => <span className="figures-tabular text-muted-foreground">{row.original.lines.length}</span>
  },
  {
    accessorKey: "status",
    header: "Status",
    cell: ({ row }) => <StatusBadge status={row.original.status} />
  },
  {
    id: "verify",
    header: () => <span className="sr-only">Verify</span>,
    cell: ({ row }) => <VerifyEntry entryId={row.original.id} />
  }
];

export default function JournalPage() {
  const { page, pageSize, filters, setFilter, setPage, setPageSize } = useTableUrlState();
  const institutions = useInstitutions();

  const apiFilter = useMemo(
    () => ({
      institutionId: filters.institutionId,
      status: filters.status,
      dateFrom: filters.dateFrom,
      dateTo: filters.dateTo,
      page,
      pageSize
    }),
    [filters, page, pageSize]
  );

  const entries = useJournalEntries(apiFilter);
  const result = entries.data;

  const institutionOptions = institutions.data?.ok
    ? institutions.data.institutions.map((i) => ({ value: i.id, label: i.name }))
    : [];

  return (
    <>
      <PageHeader
        title="Journal"
        description="Balanced double-entry postings against the chart of accounts."
        actions={<PostEntryDialog institutionOptions={institutionOptions} />}
      />
      <FilterBar>
        <SelectFilter
          label="Institution"
          value={filters.institutionId}
          options={institutionOptions}
          onChange={(value) => setFilter("institutionId", value)}
        />
        <SelectFilter
          label="Status"
          value={filters.status}
          options={[
            { value: "POSTED", label: "Posted" },
            { value: "REVERSED", label: "Reversed" }
          ]}
          onChange={(value) => setFilter("status", value)}
        />
        <DateRangeFilter
          label="Effective"
          dateFrom={filters.dateFrom}
          dateTo={filters.dateTo}
          onChange={(dateFrom, dateTo) => {
            setFilter("dateFrom", dateFrom);
            setFilter("dateTo", dateTo);
          }}
        />
      </FilterBar>
      <DataTable
        columns={columns}
        data={result?.ok ? result.entries : []}
        total={result?.ok ? result.pagination.total : 0}
        page={page}
        pageSize={pageSize}
        onPageChange={setPage}
        onPageSizeChange={setPageSize}
        isPending={entries.isPending}
        error={result?.ok === false ? result.error : undefined}
        emptyMessage="No journal entries match these filters."
        emptyIcon={BookText}
      />
    </>
  );
}

type LineDraft = JournalLine;

const emptyLine: LineDraft = { accountCode: "", direction: "DEBIT", amountMinor: 0, currency: "USD" };

function PostEntryDialog({ institutionOptions }: { institutionOptions: { value: string; label: string }[] }) {
  const [open, setOpen] = useState(false);
  const [id, setId] = useState("");
  const [institutionId, setInstitutionId] = useState("");
  const [effectiveDate, setEffectiveDate] = useState("");
  const [description, setDescription] = useState("");
  const [lines, setLines] = useState<{ accountCode: string; direction: "DEBIT" | "CREDIT"; amount: string; currency: string }[]>([
    { accountCode: "", direction: "DEBIT", amount: "", currency: "USD" },
    { accountCode: "", direction: "CREDIT", amount: "", currency: "USD" }
  ]);
  const post = usePostJournalEntry();

  const totals = lines.reduce(
    (acc, line) => {
      const minor = Math.round((Number(line.amount) || 0) * 100);
      if (line.direction === "DEBIT") acc.debit += minor;
      else acc.credit += minor;
      return acc;
    },
    { debit: 0, credit: 0 }
  );
  const balanced = totals.debit === totals.credit && totals.debit > 0;

  async function submit() {
    const payload = {
      id,
      institutionId,
      fiscalYear: effectiveDate ? Number(effectiveDate.slice(0, 4)) : new Date().getFullYear(),
      effectiveDate,
      description,
      lines: lines.map<JournalLine>((line) => ({
        accountCode: line.accountCode,
        direction: line.direction,
        amountMinor: Math.round((Number(line.amount) || 0) * 100),
        currency: line.currency.toUpperCase()
      }))
    };
    const outcome = await post.mutateAsync(payload);
    if (outcome.ok) {
      toast.success(`Entry ${id} posted`);
      setOpen(false);
      setId("");
      setDescription("");
      setLines([
        { accountCode: "", direction: "DEBIT", amount: "", currency: "USD" },
        { accountCode: "", direction: "CREDIT", amount: "", currency: "USD" }
      ]);
    } else {
      toast.error(outcome.error);
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button>
          <Plus aria-hidden />
          Post entry
        </Button>
      </DialogTrigger>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>Post journal entry</DialogTitle>
          <DialogDescription>
            Debits and credits must balance per currency before the entry can post.
          </DialogDescription>
        </DialogHeader>
        <div className="grid gap-4">
          <div className="grid gap-3 sm:grid-cols-2">
            <div className="grid gap-1.5">
              <Label htmlFor="entry-id">Entry ID</Label>
              <Input id="entry-id" className="font-mono" value={id} onChange={(e) => setId(e.target.value)} />
            </div>
            <div className="grid gap-1.5">
              <Label>Institution</Label>
              <Select value={institutionId} onValueChange={setInstitutionId}>
                <SelectTrigger aria-label="Institution">
                  <SelectValue placeholder="Select institution" />
                </SelectTrigger>
                <SelectContent>
                  {institutionOptions.map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      {option.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="grid gap-1.5">
              <Label htmlFor="entry-date">Effective date</Label>
              <Input id="entry-date" type="date" value={effectiveDate} onChange={(e) => setEffectiveDate(e.target.value)} />
            </div>
            <div className="grid gap-1.5">
              <Label htmlFor="entry-desc">Description</Label>
              <Input id="entry-desc" value={description} onChange={(e) => setDescription(e.target.value)} />
            </div>
          </div>

          <div className="grid gap-2">
            <div className="flex items-center justify-between">
              <Label>Lines</Label>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setLines([...lines, { ...emptyLine, amount: "" }])}
              >
                <Plus aria-hidden />
                Add line
              </Button>
            </div>
            {lines.map((line, index) => (
              <div key={index} className="flex flex-wrap items-center gap-2">
                <Input
                  aria-label={`Line ${index + 1} account`}
                  placeholder="Account"
                  className="w-24 font-mono"
                  value={line.accountCode}
                  onChange={(e) => updateLine(setLines, lines, index, { accountCode: e.target.value })}
                />
                <Select
                  value={line.direction}
                  onValueChange={(value) => updateLine(setLines, lines, index, { direction: value as "DEBIT" | "CREDIT" })}
                >
                  <SelectTrigger className="w-28" aria-label={`Line ${index + 1} direction`}>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="DEBIT">Debit</SelectItem>
                    <SelectItem value="CREDIT">Credit</SelectItem>
                  </SelectContent>
                </Select>
                <Input
                  aria-label={`Line ${index + 1} amount`}
                  type="number"
                  min="0"
                  step="0.01"
                  placeholder="0.00"
                  className="w-28 font-mono"
                  value={line.amount}
                  onChange={(e) => updateLine(setLines, lines, index, { amount: e.target.value })}
                />
                <Input
                  aria-label={`Line ${index + 1} currency`}
                  className="w-20 font-mono uppercase"
                  value={line.currency}
                  onChange={(e) => updateLine(setLines, lines, index, { currency: e.target.value })}
                />
                {lines.length > 2 ? (
                  <Button
                    variant="ghost"
                    size="icon"
                    aria-label={`Remove line ${index + 1}`}
                    onClick={() => setLines(lines.filter((_, i) => i !== index))}
                  >
                    <Trash2 aria-hidden />
                  </Button>
                ) : null}
              </div>
            ))}
            <div
              className={cn(
                "flex items-center justify-between rounded-md px-3 py-2 text-xs figures-tabular",
                balanced ? "bg-success/12 text-success" : "bg-muted text-muted-foreground"
              )}
            >
              <span>Debits {formatMoney("", totals.debit)}</span>
              <span>{balanced ? "Balanced" : "Not balanced"}</span>
              <span>Credits {formatMoney("", totals.credit)}</span>
            </div>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => setOpen(false)}>
            Cancel
          </Button>
          <Button onClick={submit} disabled={!balanced || !id || !institutionId || post.isPending}>
            Post entry
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function updateLine(
  setLines: React.Dispatch<React.SetStateAction<{ accountCode: string; direction: "DEBIT" | "CREDIT"; amount: string; currency: string }[]>>,
  lines: { accountCode: string; direction: "DEBIT" | "CREDIT"; amount: string; currency: string }[],
  index: number,
  patch: Partial<{ accountCode: string; direction: "DEBIT" | "CREDIT"; amount: string; currency: string }>
) {
  setLines(lines.map((line, i) => (i === index ? { ...line, ...patch } : line)));
}
