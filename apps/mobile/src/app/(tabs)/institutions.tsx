// Institutions: the public registry, with published balances per institution.
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { FlatList, Pressable, View } from "react-native";

import { publicApi } from "@/lib/api";
import { formatMoney } from "@/lib/format";
import { Body, Card, Mono, Muted, Row, StatusBadge } from "@/components/ui";
import { spacing, usePalette } from "@/theme";

export default function Institutions() {
  const palette = usePalette();
  const [expanded, setExpanded] = useState<string>("");

  const institutions = useQuery({
    queryKey: ["institutions-page"],
    queryFn: () => publicApi.institutions(1, 50),
  });
  const balances = useQuery({
    queryKey: ["balances", expanded],
    queryFn: () => publicApi.balances(expanded, 1, 50),
    enabled: expanded !== "",
  });

  return (
    <FlatList
      contentContainerStyle={{ padding: spacing.lg, gap: spacing.sm }}
      data={institutions.data?.institutions ?? []}
      keyExtractor={(institution) => institution.id}
      renderItem={({ item }) => {
        const open = expanded === item.id;
        return (
          <Pressable onPress={() => setExpanded(open ? "" : item.id)}>
            <Card style={{ gap: spacing.sm }}>
              <Row>
                <View style={{ flex: 1, marginRight: spacing.sm }}>
                  <Body style={{ fontWeight: "600" }}>{item.name}</Body>
                  <Muted>
                    {item.id} · {item.type} · {item.countryCode}
                  </Muted>
                </View>
                <StatusBadge status={item.status} />
              </Row>
              {open ? (
                <View
                  style={{
                    borderTopWidth: 1,
                    borderTopColor: palette.border,
                    paddingTop: spacing.sm,
                    gap: spacing.xs,
                  }}
                >
                  {balances.isLoading ? (
                    <Muted>Loading balances…</Muted>
                  ) : (balances.data?.balances.length ?? 0) === 0 ? (
                    <Muted>No published balances.</Muted>
                  ) : (
                    balances.data?.balances.map((balance) => (
                      <Row key={`${balance.accountCode}-${balance.currency}`}>
                        <Muted style={{ flex: 1, marginRight: spacing.sm }}>
                          {balance.accountCode} {balance.accountName}
                        </Muted>
                        <Mono>
                          {formatMoney(balance.balanceMinor, balance.currency)}
                        </Mono>
                      </Row>
                    ))
                  )}
                </View>
              ) : null}
            </Card>
          </Pressable>
        );
      }}
      ListEmptyComponent={
        <Card>
          <Muted>
            {institutions.isLoading
              ? "Loading…"
              : institutions.isError
                ? "The public API is unreachable."
                : "No institutions published."}
          </Muted>
        </Card>
      }
    />
  );
}
