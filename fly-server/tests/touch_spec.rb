require 'time'

RSpec.describe 'TOUCH' do
    context 'admin' do
        before(:all) do
            admin.write_file('touch-admin.txt', "hello\nworld\n")
            sleep 0.5
            @resp = admin.cmd('TOUCH', 'touch-admin.txt')
        end

        it 'returns OK' do
            expect(@resp).to be_ok
        end

        it 'updates modified time' do
            resp = admin.cmd!('LIST', 'touch-admin.txt')
            expect(resp).to be_a(Wire::Table)
            expect(resp.row_count).to eq(1)
            expect(resp[0][1].value).to eq('touch-admin.txt')

            mtime = Time.parse(resp[0][3].value)
            expect(mtime).to be_within(0.100).of(Time.now)
        end

        it 'creates file if not exist' do
            resp = admin.cmd('TOUCH', 'touch-admin-new.txt')
            expect(resp).to be_ok

            resp = admin.cmd!('LIST', 'touch-admin-new.txt')
            expect(resp).to be_a(Wire::Table)
            expect(resp.row_count).to eq(1)
            expect(resp[0][1].value).to eq('touch-admin-new.txt')

            mtime = Time.parse(resp[0][3].value)
            expect(mtime).to be_within(0.100).of(Time.now)
        end

        it 'returns error when directory does not exist' do
            resp = admin.cmd('TOUCH', 'some-dir-touch/touch-admin-new.txt')
            expect(resp).to be_error('NOTFOUND')
        end
    end

    context 'unauthenticated' do
        it 'returns error' do
            admin.write_file('touch-unauth.txt', "hello\nworld\n")
            resp = unauth.cmd('TOUCH', 'touch-unauth.txt')
            expect(resp).to be_error('DENIED')
        end
    end

    context 'single user' do
        before(:all) do
            single_user.write_file('touch-single-user.txt', "hello\nworld\n")
            sleep 0.5
            @resp = single_user.cmd('TOUCH', 'touch-single-user.txt')
        end

        it 'returns OK' do
            expect(@resp).to be_ok
        end

        it 'updates modified time' do
            resp = single_user.cmd!('LIST', 'touch-single-user.txt')
            expect(resp).to be_a(Wire::Table)
            expect(resp.row_count).to eq(1)
            expect(resp[0][1].value).to eq('touch-single-user.txt')

            mtime = Time.parse(resp[0][3].value)
            expect(mtime).to be_within(0.100).of(Time.now)
        end
    end
end