// Explorer: browse all published journal entries with institution filtering
// and infinite server-side pagination over the public tier.
import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { useRouter } from "expo-router";
import { useState } from "react";
import { FlatList, Pressable, ScrollView, Text, View } from "react-native";

import { publicApi } from "@/lib/api";
import { entryTotal } from "@/lib/entries";
import { formatMoney } from "@/lib/format";
import { Body, Card, Mono, Muted, Row, StatusBadge } from "@/components/ui";
import { radius, spacing, usePalette } from "@/theme";

const PAGE_SIZE = 15;

function InstitutionChips({
  selected,
  onSelect,
}: {
  selected: string;
  onSelect: (id: string) => void;
}) {
  const palette = usePalette();
  const institutions = useQuery({
    queryKey: ["institutions-all"],
    queryFn: () => publicApi.institutions(1, 50),
  });

  const chips = [{ id: "", name: "All" }, ...(institutions.data?.institutions ?? [])];
  return (
    <ScrollView
      horizontal
      showsHorizontalScrollIndicator={false}
      contentContainerStyle={{ gap: spacing.sm, padding: spacing.lg }}
    >
      {chips.map((chip) => {
        const active = selected === chip.id;
        return (
          <Pressable
            key={chip.id || "all"}
            accessibilityRole="button"
            accessibilityState={{ selected: active }}
            onPress={() => onSelect(chip.id)}
            style={{
              borderRadius: radius.pill,
              borderWidth: 1,
              borderColor: active ? palette.primary : palette.border,
              backgroundColor: active ? palette.surfaceRaised : "transparent",
              paddingHorizontal: spacing.md,
              paddingVertical: spacing.xs + 2,
            }}
          >
            <Text
              style={{
                color: active ? palette.primary : palette.textMuted,
                fontSize: 13,
                fontWeight: active ? "600" : "400",
              }}
            >
              {chip.name}
            </Text>
          </Pressable>
        );
      })}
    </ScrollView>
  );
}

export default function Explorer() {
  const router = useRouter();
  const [institutionId, setInstitutionId] = useState("");

  const entries = useInfiniteQuery({
    queryKey: ["explorer", institutionId],
    initialPageParam: 1,
    queryFn: ({ pageParam }) =>
      publicApi.journalEntries({
        page: pageParam,
        pageSize: PAGE_SIZE,
        institutionId: institutionId || undefined,
      }),
    getNextPageParam: (last) =>
      last.pagination.page * last.pagination.pageSize < last.pagination.total
        ? last.pagination.page + 1
        : undefined,
  });

  const rows = entries.data?.pages.flatMap((page) => page.entries) ?? [];
  const total = entries.data?.pages[0]?.pagination.total;

  return (
    <View style={{ flex: 1 }}>
      <InstitutionChips selected={institutionId} onSelect={setInstitutionId} />
      <FlatList
        contentContainerStyle={{
          paddingHorizontal: spacing.lg,
          paddingBottom: spacing.xl,
          gap: spacing.sm,
        }}
        ListHeaderComponent={
          <Muted style={{ marginBottom: spacing.sm }}>
            {total !== undefined ? `${total} published entries` : " "}
          </Muted>
        }
        data={rows}
        keyExtractor={(entry) => entry.id}
        onEndReached={() => {
          if (entries.hasNextPage && !entries.isFetchingNextPage) {
            void entries.fetchNextPage();
          }
        }}
        onEndReachedThreshold={0.4}
        renderItem={({ item }) => (
          <Card
            onPress={() =>
              router.push({ pathname: "/entry/[id]", params: { id: item.id } })
            }
            style={{ gap: spacing.xs }}
          >
            <Row>
              <Mono style={{ flex: 1, marginRight: spacing.sm }}>{item.id}</Mono>
              <StatusBadge status={item.status} />
            </Row>
            <Row>
              <Muted>
                {item.institutionId} · FY{item.fiscalYear} · {item.effectiveDate}
              </Muted>
              <Body style={{ fontWeight: "600" }}>
                {formatMoney(...entryTotal(item))}
              </Body>
            </Row>
          </Card>
        )}
        ListFooterComponent={
          entries.isFetchingNextPage ? (
            <Muted style={{ textAlign: "center", padding: spacing.md }}>Loading…</Muted>
          ) : null
        }
        ListEmptyComponent={
          <Card>
            <Muted>
              {entries.isLoading
                ? "Loading…"
                : entries.isError
                  ? "The public API is unreachable."
                  : "No entries match this filter."}
            </Muted>
          </Card>
        }
      />
    </View>
  );
}
