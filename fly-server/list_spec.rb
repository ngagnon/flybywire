require 'time'

RSpec.describe 'LIST' do
    context 'admin' do
        before(:all) do
            admin.cmd!('MKDIR', 'list-admin')

            @files = [
                {path: 'list-admin/file1.txt', content: "hello\nworld\n"},
                {path: 'list-admin/file2.txt', content: "hello\nworld\nfoo\nbar\n"},
                {path: 'list-admin/file3.txt', content: "hello\nworld\nfoo\nbar\nsomething\nelse\n"}
            ]

            @files.each do |f| 
                admin.write_file(f[:path], f[:content])
                f[:mtime] = Time.now
            end

            admin.cmd!('MKDIR', 'list-admin/folderthing')
        end

        describe 'folder' do
            it 'returns list of files' do
                resp = admin.cmd('LIST', 'list-admin')
                expect(resp).to be_a(Wire::Table)
                expect(resp.row_count).to eq(4)
                expect(resp.col_count).to eq(4)

                @files.each_with_index do |f, i|
                    expect(resp[i][0]).to be_a(Wire::String)
                    expect(resp[i][0].value).to eq('F')

                    expect(resp[i][1]).to be_a(Wire::String)
                    expect(resp[i][1].value).to eq(f[:path].split('/')[1])

                    expect(resp[i][2]).to be_a(Wire::Integer)
                    expect(resp[i][2].value).to eq(f[:content].length)

                    expect(resp[i][3]).to be_a(Wire::String)
                    mtime = DateTime.strptime(resp[i][3].value, '%Y-%m-%dT%H:%M:%S.%NZ')
                    expect(mtime.to_time).to be_within(0.1).of(f[:mtime])
                end

                expect(resp[3][0]).to be_a(Wire::String)
                expect(resp[3][0].value).to eq('D')

                expect(resp[3][1]).to be_a(Wire::String)
                expect(resp[3][1].value).to eq('folderthing')

                expect(resp[3][2]).to be_a(Wire::Null)

                expect(resp[3][3]).to be_a(Wire::String)
                mtime = DateTime.strptime(resp[3][3].value, '%Y-%m-%dT%H:%M:%S.%NZ')
                expect(mtime.to_time).to be_within(0.100).of(Time.now)
            end

            it 'does not show .fly' do
                resp = admin.cmd('LIST', '/')
                expect(resp).to be_a(Wire::Table)

                resp.each do |f|
                    expect(f[1].value).not_to eq('.fly')
                end
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
                mtime = DateTime.strptime(resp[0][3].value, '%Y-%m-%dT%H:%M:%S.%NZ')
                expect(mtime.to_time).to be_within(0.100).of(@files[1][:mtime])
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