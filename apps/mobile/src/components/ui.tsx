// Small shared primitives for the public monitoring app: themed surfaces,
// text styles, status badges, and list rows. Deliberately minimal — no
// component library, just the design tokens.
import type { PropsWithChildren } from "react";
import {
  Pressable,
  StyleSheet,
  Text,
  View,
  type TextStyle,
  type ViewStyle,
} from "react-native";

import { radius, spacing, usePalette } from "@/theme";

export function Card({
  children,
  style,
  onPress,
}: PropsWithChildren<{ style?: ViewStyle; onPress?: () => void }>) {
  const palette = usePalette();
  const surface: ViewStyle = {
    backgroundColor: palette.surface,
    borderColor: palette.border,
    borderWidth: StyleSheet.hairlineWidth,
    borderRadius: radius.card,
    padding: spacing.lg,
  };
  if (onPress) {
    return (
      <Pressable
        accessibilityRole="button"
        onPress={onPress}
        style={({ pressed }) => [surface, { opacity: pressed ? 0.7 : 1 }, style]}
      >
        {children}
      </Pressable>
    );
  }
  return <View style={[surface, style]}>{children}</View>;
}

export function Title({ children }: PropsWithChildren) {
  const palette = usePalette();
  return (
    <Text style={{ color: palette.text, fontSize: 22, fontWeight: "700" }}>
      {children}
    </Text>
  );
}

export function Muted({ children, style }: PropsWithChildren<{ style?: TextStyle }>) {
  const palette = usePalette();
  return (
    <Text style={[{ color: palette.textMuted, fontSize: 13 }, style]}>{children}</Text>
  );
}

export function Body({ children, style }: PropsWithChildren<{ style?: TextStyle }>) {
  const palette = usePalette();
  return (
    <Text style={[{ color: palette.text, fontSize: 15 }, style]}>{children}</Text>
  );
}

export function Mono({ children, style }: PropsWithChildren<{ style?: TextStyle }>) {
  const palette = usePalette();
  return (
    <Text
      style={[
        { color: palette.text, fontSize: 12, fontVariant: ["tabular-nums"] },
        style,
      ]}
      numberOfLines={1}
    >
      {children}
    </Text>
  );
}

export function StatusBadge({ status }: { status: string }) {
  const palette = usePalette();
  const color =
    status === "POSTED" || status === "ACTIVE"
      ? palette.success
      : status === "REVERSED" || status === "INACTIVE"
        ? palette.warning
        : palette.textMuted;
  return (
    <View
      style={{
        borderColor: color,
        borderWidth: 1,
        borderRadius: radius.pill,
        paddingHorizontal: spacing.sm,
        paddingVertical: 2,
      }}
    >
      <Text style={{ color, fontSize: 11, fontWeight: "600" }}>{status}</Text>
    </View>
  );
}

export function Row({ children, style }: PropsWithChildren<{ style?: ViewStyle }>) {
  return (
    <View
      style={[
        { flexDirection: "row", alignItems: "center", justifyContent: "space-between" },
        style,
      ]}
    >
      {children}
    </View>
  );
}
