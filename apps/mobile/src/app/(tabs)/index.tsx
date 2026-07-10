// Overview: the public pulse of the treasury — published entry volume,
// institutions, and the latest published movements.
import { useQuery } from "@tanstack/react-query";
import { useRouter } from "expo-router";
import { FlatList, RefreshControl, View } from "react-native";

import { publicApi } from "@/lib/api";
import { formatMoney } from "@/lib/format";
import { entryTotal } from "@/lib/entries";
import { Body, Card, Mono, Muted, Row, StatusBadge, Title } from "@/components/ui";
import { spacing, usePalette } from "@/theme";

export default function Overview() {
  const palette = usePalette();
  const router = useRouter();
  const entries = useQuery({
    queryKey: ["overview-entries"],
    queryFn: () => publicApi.journalEntries({ pageSize: 10 }),
  });
  const institutions = useQuery({
    queryKey: ["overview-institutions"],
    queryFn: () => publicApi.institutions(1, 1),
  });

  return (
    <FlatList
      contentContainerStyle={{ padding: spacing.lg, gap: spacing.md }}
      refreshControl={
        <RefreshControl
          refreshing={entries.isRefetching}
          onRefresh={() => {
            void entries.refetch();
            void institutions.refetch();
          }}
          tintColor={palette.textMuted}
        />
      }
      ListHeaderComponent={
        <View style={{ gap: spacing.md, marginBottom: spacing.md }}>
          <Title>Public ledger</Title>
          <Row style={{ gap: spacing.md }}>
            <Card style={{ flex: 1 }}>
              <Muted>Published entries</Muted>
              <Body style={{ fontSize: 24, fontWeight: "700" }}>
                {entries.data?.pagination.total ?? "—"}
              </Body>
            </Card>
            <Card style={{ flex: 1 }}>
              <Muted>Institutions</Muted>
              <Body style={{ fontSize: 24, fontWeight: "700" }}>
                {institutions.data?.pagination.total ?? "—"}
              </Body>
            </Card>
          </Row>
          <Muted>
            Every entry below is published under the publication policy and
            can be verified independently — open one and tap Verify.
          </Muted>
        </View>
      }
      data={entries.data?.entries ?? []}
      keyExtractor={(entry) => entry.id}
      renderItem={({ item }) => (
        <Card
          onPress={() => router.push({ pathname: "/entry/[id]", params: { id: item.id } })}
          style={{ gap: spacing.xs }}
        >
            <Row>
              <Mono style={{ flex: 1, marginRight: spacing.sm }}>{item.id}</Mono>
              <StatusBadge status={item.status} />
            </Row>
            <Row>
              <Muted>
                {item.institutionId} · {item.effectiveDate}
              </Muted>
              <Body style={{ fontWeight: "600" }}>
                {formatMoney(...entryTotal(item))}
              </Body>
            </Row>
        </Card>
      )}
      ListEmptyComponent={
        <Card>
          <Muted>
            {entries.isLoading
              ? "Loading published entries…"
              : entries.isError
                ? "The public API is unreachable. Pull to retry."
                : "No published entries yet."}
          </Muted>
        </Card>
      }
    />
  );
}
