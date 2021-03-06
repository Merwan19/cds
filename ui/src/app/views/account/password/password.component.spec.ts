/* tslint:disable:no-unused-variable */

import {TestBed, getTestBed, tick, fakeAsync, inject} from '@angular/core/testing';
import {APP_BASE_HREF} from '@angular/common';
import {MockBackend} from '@angular/http/testing';
import {XHRBackend, Response, ResponseOptions} from '@angular/http';
import {Injector} from '@angular/core';

import {UserService} from '../../../service/user/user.service';
import {AuthentificationStore} from '../../../service/auth/authentification.store';
import {AppModule} from '../../../app.module';
import {PasswordComponent} from './password.component';
import {RouterTestingModule} from '@angular/router/testing';
import {AccountModule} from '../account.module';

describe('CDS: PasswordComponent', () => {

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [],
            providers: [
                { provide: APP_BASE_HREF, useValue: '/' },
                { provide: XHRBackend, useClass: MockBackend },
                UserService,
                AuthentificationStore,
            ],
            imports : [
                AppModule,
                RouterTestingModule.withRoutes([]),
                AccountModule
            ]
        });
    });


    it('Reset Password OK', fakeAsync(  inject([XHRBackend], (backend: MockBackend) => {
        // Create loginComponent
        let fixture = TestBed.createComponent(PasswordComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        // Mock Http reset password request
        backend.connections.subscribe(connection => {
            connection.mockRespond(new Response(new ResponseOptions({})));
        });

        let compiled = fixture.debugElement.nativeElement;

        // Start detecting change in model
        fixture.detectChanges();
        tick(50);

        // Simulate user typing
        let inputUsername = compiled.querySelector('input[name="username"]');
        inputUsername.value = 'foo';
        inputUsername.dispatchEvent(new Event('input'));

        let inputEmail = compiled.querySelector('input[name="email"]');
        inputEmail.value = 'bar@foo.bar';
        inputEmail.dispatchEvent(new Event('input'));

        // Simulate user click
        compiled.querySelector('#resetPasswordButton').click();

        expect(backend.connectionsArray.length).toBe(1);
        let request: any = JSON.parse(backend.connectionsArray[0].request.getBody());
        expect(request.user.username).toBe('foo', 'Username must be foo');
        expect(request.user.email).toBe('bar@foo.bar', 'Email must be bar@foo.bar');
        expect(fixture.componentInstance.showWaitingMessage).toBeTruthy('Waiting message must be true');

    })));
});
