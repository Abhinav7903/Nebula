"use client";

import type { ReactNode } from "react";
import { Provider } from "react-redux";
import { store } from "@/store/store";

interface Props {
  children: ReactNode;
}

export default function ReduxProvider({ children }: Props) {
  return <Provider store={store}>{children}</Provider>;
}
