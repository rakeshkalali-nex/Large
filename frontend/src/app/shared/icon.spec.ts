import { TestBed } from '@angular/core/testing';
import { expect, it } from 'vitest';
import { IconComponent } from './icon';

it('renders a locally bundled decorative Lucide SVG without global icon registration', () => {
  const fixture = TestBed.createComponent(IconComponent);
  fixture.componentRef.setInput('name', 'menu');
  fixture.detectChanges();
  expect(fixture.nativeElement.getAttribute('aria-hidden')).toBe('true');
  expect(fixture.nativeElement.querySelector('svg')).not.toBeNull();
  fixture.componentRef.setInput('name', 'projects');
  fixture.detectChanges();
  expect(fixture.nativeElement.querySelector('svg')).not.toBeNull();
});
