import { TestBed } from '@angular/core/testing';
import { describe, expect, it, vi } from 'vitest';
import { PendingButtonComponent } from './pending-button';

/** WCAG relative luminance for the resolved sRGB colors used by this control. */
function luminance(color: string): number {
  const channels = color.match(/[\d.]+/g)!.slice(0, 3).map(value => {
    const channel = Number(value) / 255;
    return channel <= 0.04045 ? channel / 12.92 : ((channel + 0.055) / 1.055) ** 2.4;
  });
  return channels[0] * 0.2126 + channels[1] * 0.7152 + channels[2] * 0.0722;
}

function contrast(foreground: string, background: string): number {
  const values = [luminance(foreground), luminance(background)];
  return (Math.max(...values) + 0.05) / (Math.min(...values) + 0.05);
}

describe('shared pending button variant contrast', () => {
  it.each(['primary', 'secondary', 'destructive'] as const)('%s keeps readable, variant-specific interaction states', variant => {
    const fixture = TestBed.createComponent(PendingButtonComponent);
    fixture.componentRef.setInput('label', 'Save');
    fixture.componentRef.setInput('variant', variant);
    fixture.detectChanges();
    const button = fixture.nativeElement.querySelector('button') as HTMLButtonElement;
    const base = getComputedStyle(button);
    expect(contrast(base.color, base.backgroundColor)).toBeGreaterThanOrEqual(4.5);
    if (variant === 'secondary') expect(contrast(base.borderTopColor, base.backgroundColor)).toBeGreaterThanOrEqual(3);

    // JSDOM cannot exercise pointer pseudo-classes. Check Angular's emitted CSS
    // contract, then resolve each state's actual declarations on the real button.
    const rules = Array.from(document.styleSheets).flatMap(sheet => Array.from(sheet.cssRules))
      .filter((rule): rule is CSSStyleRule => 'selectorText' in rule);
    for (const state of ['hover', 'active']) {
      const rule = rules.find(candidate => candidate.selectorText.includes(`data-variant=${variant}`)
        && candidate.selectorText.includes(`:${state}`) && candidate.selectorText.includes(':not(:disabled)'));
      expect(rule, `${variant} needs its own enabled ${state} palette`).toBeDefined();
      button.style.cssText = rule!.style.cssText;
      const computed = getComputedStyle(button);
      expect(contrast(computed.color, computed.backgroundColor)).toBeGreaterThanOrEqual(4.5);
    }
    button.style.cssText = '';

    const activated = vi.fn();
    fixture.componentInstance.activated.subscribe(activated);
    for (const state of ['disabled', 'pending']) {
      fixture.componentRef.setInput(state, true);
      fixture.detectChanges();
      button.click();
      expect(activated).not.toHaveBeenCalled();
      expect(button.disabled).toBe(true);
      const computed = getComputedStyle(button);
      expect(contrast(computed.color, computed.backgroundColor)).toBeGreaterThanOrEqual(4.5);
      fixture.componentRef.setInput(state, false);
    }
  });
});
