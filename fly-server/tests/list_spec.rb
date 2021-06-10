require 'time'

RSpec.describe 'LIST' do
    context 'admin' do
        before(:all) do
            admin.cmd!('MKDIR', 'list-admin')
            admin.write_file('list-admin/file1.txt', "hello\nworld\n")
            admin.write_file('list-admin/file2.txt', "hello\nworld\nfoo\nbar\n")
            admin.write_file('list-admin/file3.txt', "hello\nworld\nfoo\nbar\nsomething\nelse\n")
            admin.cmd!('MKDIR', 'list-admin/folderthing')
        end

        describe 'folder' do
            it 'returns list of files' do
                resp = admin.cmd('LIST', 'list-admin')
                expect(resp).to be_a(Wire::Table)
                expect(resp.row_count).to eq(4)
                expect(resp.col_count).to eq(4)

                # @TODO: refactor in a loop...
                expect(resp[0][0]).to be_a(Wire::String)
                expect(resp[0][0].value).to eq('F')

                expect(resp[0][1]).to be_a(Wire::String)
                expect(resp[0][1].value).to eq('file1.txt')

                expect(resp[0][2]).to be_a(Wire::Integer)
                expect(resp[0][2].value).to eq(12)

                expect(resp[0][3]).to be_a(Wire::String)
                # @TODO: validate format
                mtime = Time.parse(resp[0][3].value)
                expect(mtime).to be_within(0.100).of(Time.now)

                expect(resp[1][0]).to be_a(Wire::String)
                expect(resp[1][0].value).to eq('F')

                expect(resp[1][1]).to be_a(Wire::String)
                expect(resp[1][1].value).to eq('file2.txt')

                expect(resp[1][2]).to be_a(Wire::Integer)
                expect(resp[1][2].value).to eq(20)

                expect(resp[1][3]).to be_a(Wire::String)
                # @TODO: validate format
                mtime = Time.parse(resp[1][3].value)
                expect(mtime).to be_within(0.100).of(Time.now)

                expect(resp[2][0]).to be_a(Wire::String)
                expect(resp[2][0].value).to eq('F')

                expect(resp[2][1]).to be_a(Wire::String)
                expect(resp[2][1].value).to eq('file3.txt')

                expect(resp[2][2]).to be_a(Wire::Integer)
                expect(resp[2][2].value).to eq(35)

                expect(resp[2][3]).to be_a(Wire::String)
                # @TODO: validate format
                mtime = Time.parse(resp[2][3].value)
                expect(mtime).to be_within(0.100).of(Time.now)

                expect(resp[3][0]).to be_a(Wire::String)
                expect(resp[3][0].value).to eq('D')

                expect(resp[3][1]).to be_a(Wire::String)
                expect(resp[3][1].value).to eq('folderthing')

                expect(resp[3][2]).to be_a(Wire::Null)

                expect(resp[3][3]).to be_a(Wire::String)
                # @TODO: validate format
                mtime = Time.parse(resp[3][3].value)
                expect(mtime).to be_within(0.100).of(Time.now)
            end
        end

        describe 'file' do
            it 'returns file stats' do
                resp = admin.cmd('LIST', 'list-admin/file2.txt')
                expect(resp).to be_a(Wire::Table)
                expect(resp.row_count).to eq(1)
                expect(resp.col_count).to eq(4)

                expect(resp[0][0]).to be_a(Wire::String)
                expect(resp[0][0].value).to eq('F')

                expect(resp[0][1]).to be_a(Wire::String)
                expect(resp[0][1].value).to eq('file2.txt')

                expect(resp[0][2]).to be_a(Wire::Integer)
                expect(resp[0][2].value).to eq(20)

                expect(resp[0][3]).to be_a(Wire::String)
                # @TODO: validate format
                mtime = Time.parse(resp[0][3].value)
                expect(mtime).to be_within(0.100).of(Time.now)
            end
        end
    end

    context 'unauthenticated' do
        it 'returns error' do
            admin.cmd!('MKDIR', 'list-unauth')
            admin.write_file('list-unauth/somefile.txt', "hello\nworld\n")
            resp = unauth.cmd('LIST', 'list-unauth')
            expect(resp).to be_error('DENIED')
        end
    end
end