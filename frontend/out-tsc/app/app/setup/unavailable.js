import { Component } from '@angular/core';
import * as i0 from "@angular/core";
export class UnavailablePage {
    static { this.ɵfac = function UnavailablePage_Factory(__ngFactoryType__) { return new (__ngFactoryType__ || UnavailablePage)(); }; }
    static { this.ɵcmp = /*@__PURE__*/ i0.ɵɵdefineComponent({ type: UnavailablePage, selectors: [["ng-component"]], decls: 5, vars: 0, consts: [["role", "status"]], template: function UnavailablePage_Template(rf, ctx) { if (rf & 1) {
            i0.ɵɵdomElementStart(0, "main")(1, "h1");
            i0.ɵɵtext(2, "BuildEstate");
            i0.ɵɵdomElementEnd();
            i0.ɵɵdomElementStart(3, "p", 0);
            i0.ɵɵtext(4, "This workspace is not available yet. No business operation has been performed.");
            i0.ɵɵdomElementEnd()();
        } }, encapsulation: 2 }); }
}
(() => { (typeof ngDevMode === "undefined" || ngDevMode) && i0.ɵsetClassMetadata(UnavailablePage, [{
        type: Component,
        args: [{ standalone: true, template: '<main><h1>BuildEstate</h1><p role="status">This workspace is not available yet. No business operation has been performed.</p></main>' }]
    }], null, null); })();
(() => { (typeof ngDevMode === "undefined" || ngDevMode) && i0.ɵsetClassDebugInfo(UnavailablePage, { className: "UnavailablePage", filePath: "src/app/setup/unavailable.ts", lineNumber: 3 }); })();
