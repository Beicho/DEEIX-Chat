"use client";

import * as React from "react";

import {
  normalizeSpeechText,
  resolveSpeechAction,
} from "@/features/chat/model/message-tts-utils";
import type { ChatAreaMessage } from "@/features/chat/types/messages";

type SpeechController = {
  supported: boolean;
  activeMessageKey: string | null;
  paused: boolean;
  toggleMessageSpeech: (message: ChatAreaMessage) => void;
};

function findPreferredVoice(locale: string): SpeechSynthesisVoice | null {
  if (typeof window === "undefined") {
    return null;
  }
  const voices = window.speechSynthesis.getVoices();
  const normalizedLocale = locale.toLowerCase();
  const languagePrefix = normalizedLocale.startsWith("zh") ? "zh" : normalizedLocale.split("-")[0] || "en";
  return (
    voices.find((voice) => voice.lang.toLowerCase() === normalizedLocale) ??
    voices.find((voice) => voice.lang.toLowerCase().startsWith(languagePrefix)) ??
    null
  );
}

export function useMessageSpeech(locale: string): SpeechController {
  const [supported, setSupported] = React.useState(false);
  const [activeMessageKey, setActiveMessageKey] = React.useState<string | null>(null);
  const [speaking, setSpeaking] = React.useState(false);
  const [paused, setPaused] = React.useState(false);
  const utteranceRef = React.useRef<SpeechSynthesisUtterance | null>(null);

  React.useEffect(() => {
    setSupported(typeof window !== "undefined" && "speechSynthesis" in window && "SpeechSynthesisUtterance" in window);
  }, []);

  React.useEffect(() => () => {
    if (typeof window !== "undefined" && "speechSynthesis" in window) {
      window.speechSynthesis.cancel();
    }
  }, []);

  const clearSpeechState = React.useCallback((utterance: SpeechSynthesisUtterance) => {
    if (utteranceRef.current !== utterance) {
      return;
    }
    utteranceRef.current = null;
    setActiveMessageKey(null);
    setSpeaking(false);
    setPaused(false);
  }, []);

  const toggleMessageSpeech = React.useCallback(
    (message: ChatAreaMessage) => {
      if (typeof window === "undefined" || !supported) {
        return;
      }

      const action = resolveSpeechAction({
        supported,
        speaking,
        paused,
        messageKey: message.key,
        activeMessageKey,
      });

      if (action === "unsupported") {
        return;
      }
      if (action === "pause") {
        window.speechSynthesis.pause();
        setPaused(true);
        return;
      }
      if (action === "resume") {
        window.speechSynthesis.resume();
        setPaused(false);
        return;
      }

      const text = normalizeSpeechText(message.content);
      if (!text) {
        return;
      }

      window.speechSynthesis.cancel();
      const utterance = new SpeechSynthesisUtterance(text);
      utterance.lang = locale;
      const voice = findPreferredVoice(locale);
      if (voice) {
        utterance.voice = voice;
      }
      utterance.onend = () => clearSpeechState(utterance);
      utterance.onerror = () => clearSpeechState(utterance);
      utteranceRef.current = utterance;
      setActiveMessageKey(message.key);
      setSpeaking(true);
      setPaused(false);
      window.speechSynthesis.speak(utterance);
    },
    [activeMessageKey, clearSpeechState, locale, paused, speaking, supported],
  );

  return {
    supported,
    activeMessageKey,
    paused,
    toggleMessageSpeech,
  };
}
